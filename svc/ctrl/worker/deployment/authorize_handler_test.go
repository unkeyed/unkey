package deployment_test

import (
	"context"
	"database/sql"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
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
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
)

func TestAuthorizeDeploymentStartsImageDeployment(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED)

	dep := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:    f.workspaceID,
		ProjectID:      f.projectID,
		AppID:          f.appID,
		EnvironmentID:  f.environmentID,
		Status:         mysqltype.DeploymentsStatusAwaitingApproval,
		Source:         db.DeploymentsSourceOci,
		ImageRequested: sql.NullString{String: "ghcr.io/unkey/kebap:v1", Valid: true},
	})

	actorID := uid.New("user")
	_, err := f.authorize(ctx, dep.ID, actorID)
	require.NoError(t, err)

	create := f.workflow.only(t, dep.ID)
	require.Equal(t, hydrav1.CreateDecision_CREATE_DECISION_DEPLOY, create.GetDecision())
	require.Equal(t, "ghcr.io/unkey/kebap:v1", create.GetImage().GetImage())
	require.Equal(t, ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API, create.GetTrigger().GetSource())
	require.Equal(t, actorID, create.GetTrigger().GetActor().GetId())
	require.Equal(t, mysqltype.DeploymentsStatusPending, create.statusAtCall, "Create only accepts an existing row that is already pending")

	f.requireStatus(t, ctx, dep.ID, mysqltype.DeploymentsStatusPending)
	require.Equal(t, 1, f.countAudits(t, ctx, dep.ID, actorID))
	require.Empty(t, f.github.statuses(), "an image deployment has no commit to post a status on")
}

func TestAuthorizeDeploymentStartsGitDeployment(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED)

	repo := "KEBAP/" + uid.DNS1035(8)
	require.NoError(t, f.database.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
		WorkspaceID:        f.workspaceID,
		ProjectID:          f.projectID,
		AppID:              f.appID,
		InstallationID:     rand.Int64N(1 << 40),
		RepositoryID:       rand.Int64N(1 << 40),
		RepositoryFullName: repo,
		DefaultBranch:      sql.NullString{String: "main", Valid: true},
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{},
	}))
	dep := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:            f.workspaceID,
		ProjectID:              f.projectID,
		AppID:                  f.appID,
		EnvironmentID:          f.environmentID,
		Status:                 mysqltype.DeploymentsStatusAwaitingApproval,
		Source:                 db.DeploymentsSourceGit,
		GitCommitSha:           sql.NullString{String: "9f2c1a7", Valid: true},
		ForkRepositoryFullName: sql.NullString{String: "fork/" + repo, Valid: true},
		PrNumber:               sql.NullInt64{Int64: 7, Valid: true},
	})

	actorID := uid.New("user")
	_, err := f.authorize(ctx, dep.ID, actorID)
	require.NoError(t, err)

	create := f.workflow.only(t, dep.ID)
	require.Equal(t, dep.ID, create.GetExistingDeployment().GetDeploymentId(), "Create must reuse the row the push wrote")

	f.requireStatus(t, ctx, dep.ID, mysqltype.DeploymentsStatusPending)
	require.Equal(t, 1, f.countAudits(t, ctx, dep.ID, actorID))
	require.Equal(t, []commitStatus{{repo: repo, sha: "9f2c1a7", state: "success"}}, f.github.statuses())
}

func TestAuthorizeDeploymentRejectsDeploymentNotAwaitingApproval(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED)

	dep := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.projectID,
		AppID:         f.appID,
		EnvironmentID: f.environmentID,
		Status:        mysqltype.DeploymentsStatusBuilding,
	})

	actorID := uid.New("user")
	_, err := f.authorize(ctx, dep.ID, actorID)
	require.ErrorContains(t, err, "not awaiting approval")

	require.Empty(t, f.workflow.calls(dep.ID))
	f.requireStatus(t, ctx, dep.ID, mysqltype.DeploymentsStatusBuilding)
	require.Zero(t, f.countAudits(t, ctx, dep.ID, actorID))
}

// A refused create leaves no run behind, so the row goes back to
// awaiting_approval and the approval can be retried once the cause is fixed.
func TestAuthorizeDeploymentRevertsWhenCreateRefuses(t *testing.T) {
	ctx := context.Background()
	f := newAuthorizeFixture(t, ctx, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_COMPUTE_PLAN)

	dep := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:    f.workspaceID,
		ProjectID:      f.projectID,
		AppID:          f.appID,
		EnvironmentID:  f.environmentID,
		Status:         mysqltype.DeploymentsStatusAwaitingApproval,
		Source:         db.DeploymentsSourceOci,
		ImageRequested: sql.NullString{String: "ghcr.io/unkey/kebap:v1", Valid: true},
	})

	actorID := uid.New("user")
	_, err := f.authorize(ctx, dep.ID, actorID)
	require.ErrorContains(t, err, "CREATE_OUTCOME_NO_COMPUTE_PLAN")

	require.Len(t, f.workflow.calls(dep.ID), 1)
	f.requireStatus(t, ctx, dep.ID, mysqltype.DeploymentsStatusAwaitingApproval)
	require.Zero(t, f.countAudits(t, ctx, dep.ID, actorID))
}

type authorizeFixture struct {
	database      db.Database
	seeder        *seed.Seeder
	client        *restateingress.Client
	workflow      *fakeDeployWorkflow
	github        *recordingGitHub
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func newAuthorizeFixture(t *testing.T, ctx context.Context, outcome hydrav1.CreateOutcome) *authorizeFixture {
	t.Helper()

	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	github := &recordingGitHub{Noop: githubclient.NewNoop()}
	deploymentSvc, err := deployment.New(deployment.Config{
		DB:                              database,
		Auditlogs:                       auditlogSvc,
		GitHub:                          github,
		AllowUnauthenticatedDeployments: false,
	})
	require.NoError(t, err)

	workflow := &fakeDeployWorkflow{database: database, outcome: outcome, requests: map[string][]createCall{}}
	restateCfg := containers.Restate(t,
		hydrav1.NewDeploymentServiceServer(deploymentSvc),
		hydrav1.NewDeployWorkflowServer(workflow),
	)

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID

	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "KEBAP",
		Slug:        uid.DNS1035(12),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "KEBAP",
		Slug:        uid.DNS1035(12),
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
		Kind:        mysqltype.EnvironmentKindPreview,
	})

	return &authorizeFixture{
		database:      database,
		seeder:        seeder,
		client:        restateCfg.IngressClient,
		workflow:      workflow,
		github:        github,
		workspaceID:   workspaceID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

func (f *authorizeFixture) authorize(ctx context.Context, deploymentID, actorID string) (*hydrav1.AuthorizeDeploymentResponse, error) {
	return hydrav1.NewDeploymentServiceIngressClient(f.client, deploymentID).
		AuthorizeDeployment().
		Request(ctx, &hydrav1.AuthorizeDeploymentRequest{
			Actor:         &ctrlv1.ActorInfo{Id: actorID, Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
			CorrelationId: uid.New("corr"),
		})
}

func (f *authorizeFixture) requireStatus(t *testing.T, ctx context.Context, deploymentID string, want mysqltype.DeploymentsStatus) {
	t.Helper()
	row, err := f.database.FindDeploymentById(ctx, deploymentID)
	require.NoError(t, err)
	require.Equal(t, want, row.Status)
}

func (f *authorizeFixture) countAudits(t *testing.T, ctx context.Context, deploymentID, actorID string) int {
	t.Helper()
	rows, err := f.database.ListClickhouseOutboxByWorkspace(ctx, f.workspaceID)
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

type createCall struct {
	*hydrav1.DeployCreateRequest
	statusAtCall mysqltype.DeploymentsStatus
}

// fakeDeployWorkflow stands in for DeployWorkflow so the test covers what
// AuthorizeDeployment does around Create, not Create itself.
type fakeDeployWorkflow struct {
	hydrav1.UnimplementedDeployWorkflowServer
	database db.Database
	outcome  hydrav1.CreateOutcome
	mu       sync.Mutex
	requests map[string][]createCall
}

func (w *fakeDeployWorkflow) Create(ctx restate.WorkflowSharedContext, req *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	deploymentID := restate.Key(ctx)
	row, err := w.database.FindDeploymentById(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	w.mu.Lock()
	w.requests[deploymentID] = append(w.requests[deploymentID], createCall{DeployCreateRequest: req, statusAtCall: row.Status})
	w.mu.Unlock()

	return &hydrav1.DeployCreateResponse{DeploymentId: deploymentID, Outcome: w.outcome, Detail: ""}, nil
}

func (w *fakeDeployWorkflow) calls(deploymentID string) []createCall {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.requests[deploymentID]
}

func (w *fakeDeployWorkflow) only(t *testing.T, deploymentID string) createCall {
	t.Helper()
	calls := w.calls(deploymentID)
	require.Len(t, calls, 1)
	return calls[0]
}

type commitStatus struct {
	repo  string
	sha   string
	state string
}

type recordingGitHub struct {
	*githubclient.Noop
	mu   sync.Mutex
	sent []commitStatus
}

func (g *recordingGitHub) CreateCommitStatus(_ int64, repo, sha, state, _, _, _ string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sent = append(g.sent, commitStatus{repo: repo, sha: sha, state: state})
	return nil
}

func (g *recordingGitHub) statuses() []commitStatus {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sent
}
