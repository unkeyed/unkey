package deploy_test

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/worker/githubstatus"
)

func TestGitHubStatusReportsBuildFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	database, fixture := newDeployFixture(t, ctx)
	github := &deploymentGitHub{Noop: githubclient.NewNoop()}
	require.NoError(t, database.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
		WorkspaceID: fixture.workspaceID, ProjectID: fixture.projectID, AppID: fixture.appID,
		InstallationID: 12345, RepositoryID: 67890, RepositoryFullName: fixtureRepo,
		CreatedAt: time.Now().UnixMilli(),
	}))
	row := fixture.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:          uid.New(uid.DeploymentPrefix),
		WorkspaceID: fixture.workspaceID, ProjectID: fixture.projectID,
		AppID: fixture.appID, EnvironmentID: fixture.environmentID,
		Status:       mysqltype.DeploymentsStatusPending,
		GitCommitSha: sql.NullString{Valid: true, String: fixtureCommitSHA},
	})
	metadata, err := database.FindDeploymentForDeploy(ctx, row.ID)
	require.NoError(t, err)
	audit, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	workflow, err := deploy.New(deploy.Config{
		DB: database, Auditlogs: audit, GitHub: github,
		DashboardURL: "https://app.unkey.com", DefaultDomain: "unkey.app",
	})
	require.NoError(t, err)
	cfg := containers.Restate(t,
		hydrav1.NewDeployWorkflowServer(workflow),
		hydrav1.NewGitHubStatusServiceServer(githubstatus.New(githubstatus.Config{DB: database, GitHub: github})),
	)

	invocation, err := hydrav1.NewDeployWorkflowIngressClient(cfg.IngressClient, row.ID).Submit(ctx, &hydrav1.DeployRequest{
		DeploymentId: row.ID,
		Source: &hydrav1.DeployRequest_Git{Git: &hydrav1.GitSource{
			Repository: fixtureRepo,
		}},
	})
	require.NoError(t, err)
	_, err = invocation.Attach(ctx)
	require.Error(t, err, "a build without a resolved SHA must fail before using a build provider")
	require.Eventually(t, func() bool {
		statuses := github.commitStatuses()
		return len(statuses) > 0 && statuses[len(statuses)-1].State == "failure"
	}, 10*time.Second, 50*time.Millisecond, "build failures must be visible on the commit")
	statuses := github.commitStatuses()
	require.Equal(t, "pending", statuses[0].State)
	for _, status := range statuses {
		require.Equal(t, fixtureCommitSHA, status.SHA)
		require.Equal(t, fixtureRepo, status.Repo)
		require.Equal(t, fmt.Sprintf("https://app.unkey.com/%s/projects/%s/apps/%s/deployments/%s",
			metadata.WorkspaceSlug, fixture.projectID, fixture.appID, row.ID), status.TargetURL)
		require.Equal(t, "Unkey / "+metadata.ProjectSlug+"/"+metadata.AppSlug+" - production", status.Context)
	}
	after, err := database.FindDeploymentById(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusFailed, after.Status)
}

func TestGitHubStatusCommitLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	database, fixture := newDeployFixture(t, ctx)
	github := &deploymentGitHub{Noop: githubclient.NewNoop()}
	cfg := containers.Restate(t,
		hydrav1.NewGitHubStatusServiceServer(githubstatus.New(githubstatus.Config{DB: database, GitHub: github})),
	)

	for _, scenario := range []struct {
		name string
		repo string
		sha  string
		app  string
	}{
		{name: "deployment and commit statuses", repo: fixtureRepo, sha: fixtureCommitSHA, app: "frontend"},
		{name: "another app has a separate check", repo: fixtureRepo, sha: fixtureCommitSHA, app: "worker"},
		{name: "deployment API failure does not hide the commit status", repo: "acme/forbidden", sha: fixtureCommitSHA, app: "frontend"},
		{name: "no commit means no commit status", repo: fixtureRepo, app: "frontend"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			row := fixture.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
				ID:          uid.New(uid.DeploymentPrefix),
				WorkspaceID: fixture.workspaceID, ProjectID: fixture.projectID,
				AppID: fixture.appID, EnvironmentID: fixture.environmentID,
				Status: mysqltype.DeploymentsStatusPending,
			})
			client := hydrav1.NewGitHubStatusServiceIngressClient(cfg.IngressClient, row.ID)
			logURL := fmt.Sprintf("https://app.unkey.com/steamsets/projects/%s/apps/%s/deployments/%s",
				fixture.projectID, fixture.appID, row.ID)
			_, err := client.Init().Request(ctx, &hydrav1.GitHubStatusInitRequest{
				InstallationId: 12345, Repo: scenario.repo, CommitSha: scenario.sha,
				ProjectSlug: "backend", AppSlug: scenario.app, EnvSlug: "production",
				EnvironmentLabel: "backend/" + scenario.app + " - production",
				LogUrl:           logURL, EnvironmentUrl: "https://backend-frontend-production-steamsets.unkey.app",
				IsProduction: true,
			})
			require.NoError(t, err)

			want := []string{"pending"}
			for _, transition := range []struct {
				state hydrav1.GitHubDeploymentState
				want  string
			}{
				{hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_QUEUED, "pending"},
				{hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_IN_PROGRESS, "pending"},
				{hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_SUCCESS, "success"},
				{hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_FAILURE, "failure"},
				{hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_ERROR, "error"},
				{hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_INACTIVE, "success"},
			} {
				_, err = client.ReportStatus().Request(ctx, &hydrav1.GitHubStatusReportRequest{
					State: transition.state, Description: "Deployment status changed",
				})
				require.NoError(t, err)
				want = append(want, transition.want)
			}

			var states []string
			for _, status := range github.commitStatuses() {
				if status.TargetURL != logURL {
					continue
				}
				require.Equal(t, scenario.repo, status.Repo)
				require.Equal(t, scenario.sha, status.SHA)
				require.Equal(t, "Unkey / backend/"+scenario.app+" - production", status.Context)
				states = append(states, status.State)
			}
			if scenario.sha == "" {
				require.Empty(t, states)
			} else {
				require.Equal(t, want, states)
			}
		})
	}
}

type deploymentCommitStatus struct {
	Repo        string
	SHA         string
	State       string
	TargetURL   string
	Description string
	Context     string
}

type deploymentGitHub struct {
	*githubclient.Noop
	mu       sync.Mutex
	statuses []deploymentCommitStatus
}

func (g *deploymentGitHub) CreateDeployment(_ int64, repo, _, _, _ string, _ bool) (int64, error) {
	if repo == "acme/forbidden" {
		return 0, restate.TerminalErrorf("GitHub deployments permission denied")
	}
	return 1, nil
}

func (g *deploymentGitHub) CreateDeploymentStatus(_ int64, _ string, _ int64, _, _, _, _ string) error {
	return nil
}

func (g *deploymentGitHub) CreateCommitStatus(_ int64, repo, sha, state, targetURL, description, statusContext string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.statuses = append(g.statuses, deploymentCommitStatus{
		Repo: repo, SHA: sha, State: state, TargetURL: targetURL, Description: description, Context: statusContext,
	})
	return nil
}

func (g *deploymentGitHub) commitStatuses() []deploymentCommitStatus {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.statuses)
}
