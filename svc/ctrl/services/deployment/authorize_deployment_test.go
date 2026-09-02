package deployment

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/githubstatus"
)

// TestAuthorizeDeploymentDispatchesAuditsAndUnblocksGitHub covers the full
// authorize contract: the CAS flips the row to pending, Deploy is dispatched
// and its invocation id persisted, ctrl writes the audit entry, and the blocking
// commit status is replaced through GitHubStatusService, which owns the status
// context string both writes share.
func TestAuthorizeDeploymentDispatchesAuditsAndUnblocksGitHub(t *testing.T) {
	ctx := context.Background()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID

	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "KEBAP",
		Slug:        testSlug(uid.ProjectPrefix),
	})
	environmentID := uid.New(uid.EnvironmentPrefix)
	appID := uid.New(uid.AppPrefix)
	seeder.CreateAppWithSettings(ctx, seed.CreateAppRequest{
		ID:            appID,
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "KEBAP",
		Slug:          testSlug(uid.AppPrefix),
		DefaultBranch: "main",
	}, environmentID)
	seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          environmentID,
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       appID,
		Slug:        "preview",
		Kind:        mysqltype.EnvironmentKindPreview,
	})

	const repoFullName = "kebap/kebap"
	const installationID = int64(4242)
	require.NoError(t, database.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspaceID,
		ProjectID:          project.ID,
		AppID:              appID,
		InstallationID:     installationID,
		RepositoryID:       4242,
		RepositoryFullName: repoFullName,
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{Valid: false},
	}))

	commitSHA := strings.ToLower(strings.ReplaceAll(uid.New("sha"), "_", ""))
	deployment := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         appID,
		EnvironmentID: environmentID,
		Status:        mysqltype.DeploymentsStatusAwaitingApproval,
		GitCommitSha:  sql.NullString{Valid: true, String: commitSHA},
		GitBranch:     sql.NullString{Valid: true, String: "main"},
	})

	deploys := &deployRecorder{}
	github := &commitStatusRecorder{Noop: githubclient.NewNoop()}
	restateCfg := containers.Restate(t,
		hydrav1.NewDeployServiceServer(deploys),
		// Bound with the same options as production (worker run.go). With the
		// ingress-private flag set, this RPC's Send is refused and the status
		// never arrives.
		hydrav1.NewGitHubStatusServiceServer(githubstatus.New(githubstatus.Config{
			GitHub:                          github,
			DB:                              database,
			AllowUnauthenticatedDeployments: false,
		})),
	)

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	svc := New(Config{
		Database:          database,
		Restate:           restateCfg.IngressClient,
		RestateAdmin:      nil,
		Auditlogs:         auditlogSvc,
		Bearer:            testBearer,
		EnforceDeployGate: false,
	})

	actorID := uid.New("user")
	req := connect.NewRequest(&ctrlv1.AuthorizeDeploymentRequest{
		DeploymentId: deployment.ID,
		Actor:        &ctrlv1.ActorInfo{Id: actorID, Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err = svc.AuthorizeDeployment(ctx, req)
	require.NoError(t, err)

	after, err := database.FindDeploymentById(ctx, deployment.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusPending, after.Status)
	require.True(t, after.InvocationID.Valid && after.InvocationID.String != "",
		"the invocation id must be persisted so a later cancel can reach the build")

	require.Equal(t, 1, countAuthorizeAudits(t, ctx, database, workspaceID, deployment.ID, actorID))

	// The commit status travels through GitHubStatusService as a Send, so it
	// lands after the RPC returns.
	require.Eventually(t, func() bool {
		state, repo, sha, ok := github.last()
		return ok && state == "success" && repo == repoFullName && sha == commitSHA
	}, 30*time.Second, 100*time.Millisecond,
		"the authorization commit status must replace the blocking one via GitHubStatusService")

	require.Eventually(t, func() bool { return deploys.received(deployment.ID) },
		30*time.Second, 100*time.Millisecond, "Deploy must be dispatched for the authorized row")
}

// TestAuthorizeDeploymentRequiresComputePlan covers the billing gate: a
// workspace with no Compute plan cannot start a build. The gate runs before the
// status CAS, so a blocked call leaves the deployment authorizable once the
// workspace subscribes.
func TestAuthorizeDeploymentRequiresComputePlan(t *testing.T) {
	ctx := context.Background()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID

	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "KEBAP",
		Slug:        testSlug(uid.ProjectPrefix),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "KEBAP",
		Slug:          testSlug(uid.AppPrefix),
		DefaultBranch: "main",
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
		Kind:        mysqltype.EnvironmentKindPreview,
	})
	deployment := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusAwaitingApproval,
	})

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	// The seeded workspace has a billing row with no plan and no override, and
	// Restate stays nil: the gate has to reject before anything is dispatched.
	svc := New(Config{
		Database:          database,
		Restate:           nil,
		RestateAdmin:      nil,
		Auditlogs:         auditlogSvc,
		Bearer:            testBearer,
		EnforceDeployGate: true,
	})

	req := connect.NewRequest(&ctrlv1.AuthorizeDeploymentRequest{
		DeploymentId: deployment.ID,
		Actor:        &ctrlv1.ActorInfo{Id: uid.New("user"), Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err = svc.AuthorizeDeployment(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "err: %v", err)
	require.ErrorContains(t, err, "no active Compute plan")

	after, err := database.FindDeploymentById(ctx, deployment.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusAwaitingApproval, after.Status,
		"a blocked authorization must not consume the awaiting_approval state")
}

func countAuthorizeAudits(t *testing.T, ctx context.Context, database db.Database, workspaceID, deploymentID, actorID string) int {
	t.Helper()
	rows, err := database.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		payload := string(row.Payload)
		if strings.Contains(payload, string(auditlog.DeploymentAuthorizeEvent)) &&
			strings.Contains(payload, deploymentID) &&
			strings.Contains(payload, actorID) {
			count++
		}
	}
	return count
}

// deployRecorder accepts Deploy sends without building anything.
type deployRecorder struct {
	hydrav1.UnimplementedDeployServiceServer
	mu  sync.Mutex
	ids []string
}

func (r *deployRecorder) Deploy(_ restate.ObjectContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ids = append(r.ids, req.GetDeploymentId())
	return &hydrav1.DeployResponse{}, nil
}

func (r *deployRecorder) received(deploymentID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range r.ids {
		if id == deploymentID {
			return true
		}
	}
	return false
}

// commitStatusRecorder is the noop GitHub client with CreateCommitStatus
// recording what the worker posted.
type commitStatusRecorder struct {
	*githubclient.Noop
	mu    sync.Mutex
	state string
	repo  string
	sha   string
	set   bool
}

func (c *commitStatusRecorder) CreateCommitStatus(_ int64, repo, sha, state, _, _, _ string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state, c.repo, c.sha, c.set = state, repo, sha, true
	return nil
}

func (c *commitStatusRecorder) last() (state, repo, sha string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state, c.repo, c.sha, c.set
}
