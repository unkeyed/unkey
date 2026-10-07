package deployment

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
)

const (
	authorizeBearer     = "KEBAP"
	authorizeRepo       = "acme/api"
	authorizeForkRepo   = "contributor/api"
	authorizeInstallID  = 12345
	authorizePRNumber   = 42
	authorizeDockerfile = "svc/api/Dockerfile"
	authorizeContext    = "svc/api"
	authorizeBuildCmd   = "pnpm build"
)

// TestAuthorizeDeploymentForkPRStartsTheRun pins the whole DeployRequest a fork
// PR approval hands to the run. Every field here decides what gets built:
// pr_number in particular selects refs/pull/<n>/head from the base repository
// over cloning the fork directly, so a drop would build different code.
func TestAuthorizeDeploymentForkPRStartsTheRun(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx)

	commitSHA := testCommitSHA()
	deployment := f.seedAwaitingApproval(ctx, seed.CreateDeploymentRequest{
		GitCommitSha:           sql.NullString{Valid: true, String: commitSHA},
		GitBranch:              sql.NullString{Valid: true, String: "feature/kebap"},
		GitCommitMessage:       sql.NullString{Valid: true, String: "feat: KEBAP"},
		PrNumber:               sql.NullInt64{Valid: true, Int64: authorizePRNumber},
		ForkRepositoryFullName: sql.NullString{Valid: true, String: authorizeForkRepo},
	})

	_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
	require.NoError(t, err)

	sent := f.deploys.await(t, deployment.ID)
	require.Equal(t, deployment.ID, sent.GetDeploymentId())
	require.Empty(t, sent.GetCommand())

	git := sent.GetGit()
	require.NotNil(t, git, "a fork PR approval must build from git")
	require.Equal(t, int64(authorizeInstallID), git.GetInstallationId())
	require.Equal(t, authorizeRepo, git.GetRepository(), "the build clones the base repo, not the fork")
	require.Equal(t, authorizeForkRepo, git.GetForkRepository())
	require.Equal(t, int64(authorizePRNumber), git.GetPrNumber(),
		"pr_number selects refs/pull/<n>/head over cloning the fork")
	require.Equal(t, commitSHA, git.GetCommitSha())
	require.Equal(t, "feature/kebap", git.GetBranch())
	require.Equal(t, authorizeContext, git.GetContextPath())
	require.Equal(t, authorizeDockerfile, git.GetDockerfilePath())
	require.Equal(t, authorizeBuildCmd, git.GetBuildCommand())

	after, err := f.database.FindDeploymentById(ctx, deployment.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusPending, after.Status)
	require.True(t, after.InvocationID.Valid, "the invocation id has to be persisted so a cancel can reach it")
	require.NotEmpty(t, after.InvocationID.String)

	status := f.github.await(t, commitSHA)
	require.Equal(t, "success", status.state)
	require.Equal(t, githubclient.DeployAuthorizationContext, status.context)
	require.Equal(t, authorizeRepo, status.repo)
}

// TestAuthorizeDeploymentOCIStartsTheRun pins the image arm. An awaiting
// approval row has never built, so image_requested is the only reference it
// carries and image_resolved is still null.
func TestAuthorizeDeploymentOCIStartsTheRun(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx)

	deployment := f.seedAwaitingApproval(ctx, seed.CreateDeploymentRequest{})
	f.setImageRequested(ctx, deployment.ID, "index.docker.io/library/nginx:1.27")

	_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
	require.NoError(t, err)

	sent := f.deploys.await(t, deployment.ID)
	require.Nil(t, sent.GetGit())
	require.Equal(t, "index.docker.io/library/nginx:1.27", sent.GetOciImage().GetImage())

	after, err := f.database.FindDeploymentById(ctx, deployment.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusPending, after.Status)
}

// TestAuthorizeDeploymentRefusals pins every path that must not start a run.
// Each one leaves the row where it found it, so the approve button stays.
func TestAuthorizeDeploymentRefusals(t *testing.T) {
	ctx := context.Background()

	t.Run("a deployment that is not awaiting approval is refused", func(t *testing.T) {
		f := newAuthorizeFixture(t, ctx)
		deployment := f.seedGitAwaitingApproval(ctx)
		f.setStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusReady)

		_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		f.requireNoDeploy(t, deployment.ID)
		f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusReady)
	})

	t.Run("an unknown deployment is not found", func(t *testing.T) {
		f := newAuthorizeFixture(t, ctx)

		_, err := f.svc.AuthorizeDeployment(ctx, f.request(uid.New(uid.DeploymentPrefix)))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})

	t.Run("an empty deployment id is rejected", func(t *testing.T) {
		f := newAuthorizeFixture(t, ctx)

		_, err := f.svc.AuthorizeDeployment(ctx, f.request(""))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("a missing bearer is unauthenticated", func(t *testing.T) {
		f := newAuthorizeFixture(t, ctx)
		deployment := f.seedGitAwaitingApproval(ctx)

		req := connect.NewRequest(&ctrlv1.AuthorizeDeploymentRequest{DeploymentId: deployment.ID})
		_, err := f.svc.AuthorizeDeployment(ctx, req)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		f.requireNoDeploy(t, deployment.ID)
		f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusAwaitingApproval)
	})

	t.Run("a wrong bearer is unauthenticated", func(t *testing.T) {
		f := newAuthorizeFixture(t, ctx)
		deployment := f.seedGitAwaitingApproval(ctx)

		req := connect.NewRequest(&ctrlv1.AuthorizeDeploymentRequest{DeploymentId: deployment.ID})
		req.Header().Set("Authorization", "Bearer not-"+authorizeBearer)
		_, err := f.svc.AuthorizeDeployment(ctx, req)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		f.requireNoDeploy(t, deployment.ID)
	})

	t.Run("a workspace with no compute plan is refused", func(t *testing.T) {
		f := newAuthorizeFixture(t, ctx)
		deployment := f.seedGitAwaitingApproval(ctx)
		f.clearComputePlan(ctx)

		_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
		require.Error(t, err)
		f.requireNoDeploy(t, deployment.ID)
		f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusAwaitingApproval)
	})

	t.Run("a spend suspended workspace is refused", func(t *testing.T) {
		f := newAuthorizeFixture(t, ctx)
		deployment := f.seedGitAwaitingApproval(ctx)
		f.suspendSpend(ctx)

		_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
		require.Error(t, err)
		f.requireNoDeploy(t, deployment.ID)
		f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusAwaitingApproval)
	})

	t.Run("an OCI row with no image is refused", func(t *testing.T) {
		f := newAuthorizeFixture(t, ctx)
		deployment := f.seedAwaitingApproval(ctx, seed.CreateDeploymentRequest{})

		_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		f.requireNoDeploy(t, deployment.ID)
		f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusAwaitingApproval)
	})
}

// TestAuthorizeDeploymentRevertsWhenTheRunCannotStart pins the compensation: a
// dispatch that never lands has to hand the row back so the approve button
// returns, rather than stranding it in pending with no workflow.
func TestAuthorizeDeploymentRevertsWhenTheRunCannotStart(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx)
	deployment := f.seedGitAwaitingApproval(ctx)

	f.svc.restate = restateingress.NewClient("http://127.0.0.1:1")

	_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
	require.Error(t, err)
	f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusAwaitingApproval)
}

// TestAuthorizeDeploymentRevertsWhenCreateRefuses covers the compensation this
// path depends on. ensureWorkspaceCanDeploy checks plan and spend only, so a
// target that stopped being deployable is caught by Create after the row is
// already pending, and only the revert hands the approve button back.
func TestAuthorizeDeploymentRevertsWhenCreateRefuses(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx)
	deployment := f.seedGitAwaitingApproval(ctx)

	// Nowhere left to schedule. The pre-CAS gate does not look at regions
	f.exec(ctx, "DELETE FROM app_regional_settings WHERE app_id = ?", f.appID)

	_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	f.requireNoDeploy(t, deployment.ID)
	f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusAwaitingApproval)
}

// TestRevertAuthorizationLeavesANonPendingRow pins the guard on the swap. A
// cancel that lands while the dispatch is in flight has to survive it.
func TestRevertAuthorizationLeavesANonPendingRow(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx)
	deployment := f.seedGitAwaitingApproval(ctx)
	f.setStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusCancelled)

	f.svc.revertAuthorization(ctx, deployment.ID)

	f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusCancelled)
}

// TestRevertAuthorizationLeavesAStartedRun pins the other half of the guard.
// Create sends Deploy before it persists the invocation id and cancels
// siblings, and an error from either still fails the call, so a revert must
// not re-arm approval underneath a build that is already running.
func TestRevertAuthorizationLeavesAStartedRun(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx)
	deployment := f.seedGitAwaitingApproval(ctx)
	f.setStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusPending)
	f.exec(ctx, "UPDATE deployments SET invocation_id = ? WHERE id = ?",
		uid.New("inv"), deployment.ID)

	f.svc.revertAuthorization(ctx, deployment.ID)

	f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusPending)
}

// TestAuthorizeDeploymentRefusesAnUntaggedImage pins the narrowing the image
// arm brings. Create normalizes the reference where the old path passed it
// through, so an implicit tag is refused now, and the refusal has to leave the
// approve button rather than strand the row.
func TestAuthorizeDeploymentRefusesAnUntaggedImage(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx)
	deployment := f.seedAwaitingApproval(ctx, seed.CreateDeploymentRequest{})
	f.setImageRequested(ctx, deployment.ID, "nginx")

	_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	f.requireNoDeploy(t, deployment.ID)
	f.requireStatus(ctx, deployment.ID, mysqltype.DeploymentsStatusAwaitingApproval)
}

// TestAuthorizeDeploymentIgnoresANewerSibling pins RequireLatest staying false.
// A reviewer approving a stale commit is the normal case, and refusing it
// because the branch moved on would strand the approval.
func TestAuthorizeDeploymentIgnoresANewerSibling(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx)
	deployment := f.seedGitAwaitingApproval(ctx)
	f.exec(ctx, "UPDATE deployments SET created_at = ? WHERE id = ?",
		time.Now().Add(-2*time.Hour).UnixMilli(), deployment.ID)

	newer := f.seedAwaitingApproval(ctx, seed.CreateDeploymentRequest{
		CreatedAt:        time.Now().Add(-1 * time.Hour).UnixMilli(),
		GitCommitSha:     sql.NullString{Valid: true, String: testCommitSHA()},
		GitBranch:        sql.NullString{Valid: true, String: "feature/kebap"},
		GitCommitMessage: sql.NullString{Valid: true, String: "feat: KEBAP again"},
	})

	_, err := f.svc.AuthorizeDeployment(ctx, f.request(deployment.ID))
	require.NoError(t, err)

	require.NotNil(t, f.deploys.await(t, deployment.ID))
	f.requireStatus(ctx, newer.ID, mysqltype.DeploymentsStatusAwaitingApproval)
}

type authorizeFixture struct {
	t        *testing.T
	database db.Database
	seeder   *seed.Seeder
	svc      *Service
	deploys  *authorizeDeployRecorder
	github   *recordingGitHub

	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func newAuthorizeFixture(t *testing.T, ctx context.Context) *authorizeFixture {
	t.Helper()

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
		Slug:        authorizeSlug(uid.ProjectPrefix),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "KEBAP",
		Slug:        authorizeSlug(uid.AppPrefix),
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})

	region := seeder.CreateRegion(ctx, seed.CreateRegionRequest{Name: "kebap-1", Platform: "k8s"})
	require.NoError(t, database.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		RegionID:      region.ID,
		Replicas:      1,
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
	}))

	require.NoError(t, database.UpsertAppBuildSettings(ctx, db.UpsertAppBuildSettingsParams{
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Dockerfile:    sql.NullString{Valid: true, String: authorizeDockerfile},
		DockerContext: authorizeContext,
		BuildCommand:  sql.NullString{Valid: true, String: authorizeBuildCmd},
		WatchPaths:    nil,
		AutoDeploy:    true,
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
	}))

	require.NoError(t, database.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspaceID,
		ProjectID:          project.ID,
		AppID:              app.ID,
		InstallationID:     authorizeInstallID,
		RepositoryID:       67890,
		RepositoryFullName: authorizeRepo,
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{Valid: false, Int64: 0},
	}))

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	workflow, err := deploy.New(deploy.Config{
		DB:            database,
		Auditlogs:     auditlogSvc,
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.local",
		Vault:         nil,
		GitHub:        githubclient.NewNoop(),
		Build: deploy.BuildConfig{
			Backend:    deploy.BuildBackendDepot,
			Depot:      deploy.DepotConfig{APIUrl: "", ProjectRegion: "", ProjectPrefix: "builds-test"},
			Kubernetes: deploy.KubernetesBuildConfig{Namespace: "", Image: ""},
		},
		K8s:                             nil,
		RegistryConfig:                  deploy.RegistryConfig{Repository: "", Username: "", Password: "", Insecure: false},
		BuildPlatform:                   deploy.BuildPlatform{Platform: "", Architecture: ""},
		Clickhouse:                      nil,
		BuildSteps:                      batch.NewNoop[schema.BuildStepV1](),
		BuildStepLogs:                   batch.NewNoop[schema.BuildStepLogV1](),
		AllowUnauthenticatedDeployments: false,
		RestateAdmin:                    nil,
	})
	require.NoError(t, err)

	deploys := &authorizeDeployRecorder{
		Workflow: workflow,
		requests: make(map[string]*hydrav1.DeployRequest),
	}
	restateCfg := containers.Restate(t, hydrav1.NewDeployWorkflowServer(deploys))

	github := &recordingGitHub{
		Noop:     githubclient.NewNoop(),
		statuses: make(map[string]commitStatus),
	}

	f := &authorizeFixture{
		t:        t,
		database: database,
		seeder:   seeder,
		deploys:  deploys,
		github:   github,
		svc: New(Config{
			Database:     database,
			Auditlogs:    auditlogSvc,
			Restate:      restateCfg.IngressClient,
			RestateAdmin: nil,
			GitHub:       github,
			Bearer:       authorizeBearer,
		}),
		workspaceID:   workspaceID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
	f.grantComputePlan(ctx)
	return f
}

func (f *authorizeFixture) request(deploymentID string) *connect.Request[ctrlv1.AuthorizeDeploymentRequest] {
	f.t.Helper()
	req := connect.NewRequest(&ctrlv1.AuthorizeDeploymentRequest{DeploymentId: deploymentID})
	req.Header().Set("Authorization", "Bearer "+authorizeBearer)
	return req
}

func (f *authorizeFixture) seedAwaitingApproval(ctx context.Context, req seed.CreateDeploymentRequest) db.Deployment {
	f.t.Helper()
	req.ID = uid.New(uid.DeploymentPrefix)
	req.WorkspaceID = f.workspaceID
	req.ProjectID = f.projectID
	req.AppID = f.appID
	req.EnvironmentID = f.environmentID
	req.Status = mysqltype.DeploymentsStatusAwaitingApproval
	return f.seeder.CreateDeployment(ctx, req)
}

func (f *authorizeFixture) seedGitAwaitingApproval(ctx context.Context) db.Deployment {
	f.t.Helper()
	return f.seedAwaitingApproval(ctx, seed.CreateDeploymentRequest{
		GitCommitSha:           sql.NullString{Valid: true, String: testCommitSHA()},
		GitBranch:              sql.NullString{Valid: true, String: "feature/kebap"},
		GitCommitMessage:       sql.NullString{Valid: true, String: "feat: KEBAP"},
		PrNumber:               sql.NullInt64{Valid: true, Int64: authorizePRNumber},
		ForkRepositoryFullName: sql.NullString{Valid: true, String: authorizeForkRepo},
	})
}

func (f *authorizeFixture) exec(ctx context.Context, query string, args ...any) {
	f.t.Helper()
	_, err := f.database.RW().ExecContext(ctx, query, args...)
	require.NoError(f.t, err)
}

func (f *authorizeFixture) setImageRequested(ctx context.Context, deploymentID, image string) {
	f.t.Helper()
	f.exec(ctx, "UPDATE deployments SET image_requested = ? WHERE id = ?", image, deploymentID)
}

func (f *authorizeFixture) setStatus(ctx context.Context, deploymentID string, status mysqltype.DeploymentsStatus) {
	f.t.Helper()
	f.exec(ctx, "UPDATE deployments SET status = ? WHERE id = ?", string(status), deploymentID)
}

func (f *authorizeFixture) grantComputePlan(ctx context.Context) {
	f.t.Helper()
	f.exec(ctx, "UPDATE workspace_billing SET plan_override = ? WHERE workspace_id = ?", "starter", f.workspaceID)
}

func (f *authorizeFixture) clearComputePlan(ctx context.Context) {
	f.t.Helper()
	f.exec(ctx, "UPDATE workspace_billing SET plan = NULL, plan_override = NULL WHERE workspace_id = ?", f.workspaceID)
}

func (f *authorizeFixture) suspendSpend(ctx context.Context) {
	f.t.Helper()
	f.exec(ctx, "UPDATE workspace_billing SET spend_suspended = 1 WHERE workspace_id = ?", f.workspaceID)
}

func (f *authorizeFixture) requireStatus(ctx context.Context, deploymentID string, want mysqltype.DeploymentsStatus) {
	f.t.Helper()
	row, err := f.database.FindDeploymentById(ctx, deploymentID)
	require.NoError(f.t, err)
	require.Equal(f.t, want, row.Status)
}

// requireNoDeploy proves no run started. The window has to outlast the ingress
// round trip a passing case would need.
func (f *authorizeFixture) requireNoDeploy(t *testing.T, deploymentID string) {
	t.Helper()
	require.Never(t, func() bool {
		return f.deploys.get(deploymentID) != nil
	}, 3*time.Second, 100*time.Millisecond)
}

// authorizeDeployRecorder is the real workflow with Deploy replaced, so a run
// is observed without building. Keeping the real Create is what lets these
// assertions survive the approval path moving onto it.
type authorizeDeployRecorder struct {
	*deploy.Workflow
	mu       sync.Mutex
	requests map[string]*hydrav1.DeployRequest
}

func (r *authorizeDeployRecorder) Deploy(_ restate.WorkflowContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests[req.GetDeploymentId()] = req
	return &hydrav1.DeployResponse{}, nil
}

func (r *authorizeDeployRecorder) get(deploymentID string) *hydrav1.DeployRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests[deploymentID]
}

func (r *authorizeDeployRecorder) await(t *testing.T, deploymentID string) *hydrav1.DeployRequest {
	t.Helper()
	require.Eventually(t, func() bool {
		return r.get(deploymentID) != nil
	}, 30*time.Second, 100*time.Millisecond, "the approval never reached Deploy")
	return r.get(deploymentID)
}

type commitStatus struct {
	repo    string
	state   string
	context string
}

type recordingGitHub struct {
	*githubclient.Noop
	mu       sync.Mutex
	statuses map[string]commitStatus
}

func (g *recordingGitHub) CreateCommitStatus(_ int64, repo, sha, state, _, _, statusContext string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.statuses[sha] = commitStatus{repo: repo, state: state, context: statusContext}
	return nil
}

func (g *recordingGitHub) await(t *testing.T, sha string) commitStatus {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	status, ok := g.statuses[sha]
	require.True(t, ok, "no commit status was posted for %s", sha)
	return status
}

func testCommitSHA() string {
	raw := make([]byte, 20)
	_, err := rand.Read(raw)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(raw)
}

func authorizeSlug(prefix uid.Prefix) string {
	return strings.ToLower(strings.ReplaceAll(uid.New(prefix), "_", "-"))
}
