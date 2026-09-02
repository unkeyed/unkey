package deploy_test

import (
	"context"
	"database/sql"
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
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
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

const (
	// A commit skips the GitHub lookup only when both the sha and the message
	// are present. Every fixture sends both.
	fixtureCommitSHA     = "9f2c1a7"
	fixtureCommitMessage = "add the KEBAP endpoint"
	fixtureImage         = "ghcr.io/unkey/kebap:v1"
	fixtureRepo          = "acme/api"
)

// TestCreateWritesRowAndStartsDeploy is the happy path. The recorded
// invocation id is what makes the deployment cancellable afterwards.
func TestCreateWritesRowAndStartsDeploy(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})

	deploymentID := uid.New(uid.DeploymentPrefix)
	resp := h.create(t, ctx, deploymentID, h.imageRequest())

	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, resp.GetOutcome())

	row := h.deployment(t, ctx, deploymentID)
	require.Equal(t, mysqltype.DeploymentsStatusPending, row.Status)
	require.Equal(t, h.appID, row.AppID)
	require.Equal(t, h.environmentID, row.EnvironmentID)
	require.Equal(t, db.DeploymentsTriggerApi, row.Trigger)

	step := h.queuedStep(t, ctx, deploymentID)
	require.Nil(t, step, "the queued step must still be open when Deploy has not run")

	sent := h.awaitDeploy(t, deploymentID)
	image, ok := sent.GetSource().(*hydrav1.DeployRequest_DockerImage)
	require.True(t, ok, "an image source must reach Deploy as an image")
	require.Equal(t, fixtureImage, image.DockerImage.GetImage())

	require.Eventually(t, func() bool {
		current := h.deployment(t, ctx, deploymentID)
		return current.InvocationID.Valid && current.InvocationID.String != ""
	}, 15*time.Second, 100*time.Millisecond, "the create must record the Deploy invocation id")

	require.Equal(t, 1, h.countAudits(t, ctx, auditlog.DeploymentCreateEvent, deploymentID))
}

// TestCreateReplaysAnExistingRow pins the permanent half of idempotency.
// Restate replays an invocation key only inside its retention window. Past
// that, the existing row is the only thing stopping a repeated create from
// deploying twice.
func TestCreateReplaysAnExistingRow(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})

	deploymentID := uid.New(uid.DeploymentPrefix)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
		h.create(t, ctx, deploymentID, h.imageRequest()).GetOutcome())

	first := h.deployment(t, ctx, deploymentID)
	h.awaitDeploy(t, deploymentID)
	h.deploys.forget(deploymentID)

	// A different payload under the same id must still answer with the original
	// row instead of overwriting it.
	second := h.imageRequest()
	second.Source = &hydrav1.DeployCreateRequest_Image{
		Image: &hydrav1.CreateImageSource{Image: "ghcr.io/unkey/other:v9"},
	}
	resp := h.create(t, ctx, deploymentID, second)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_REPLAYED, resp.GetOutcome())

	after := h.deployment(t, ctx, deploymentID)
	require.Equal(t, first.CreatedAt, after.CreatedAt, "a replay must not restamp the row")
	require.Equal(t, 1, h.countDeployments(t, ctx), "a replay must not write a second row")
	require.Equal(t, 1, h.countAudits(t, ctx, auditlog.DeploymentCreateEvent, deploymentID),
		"a replay must not audit a create that did not happen")

	h.requireNoDeploy(t, deploymentID)
}

// TestCreateRefusesAForeignRow covers a collision that derived ids rule out:
// the id hashes the workspace, app and environment. If one ever collided,
// answering with the row that is there would hand the caller someone else's
// deployment.
func TestCreateRefusesAForeignRow(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})

	other := h.newApp(t, ctx)
	deploymentID := uid.New(uid.DeploymentPrefix)
	h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            deploymentID,
		WorkspaceID:   h.workspaceID,
		ProjectID:     h.projectID,
		AppID:         other.appID,
		EnvironmentID: other.environmentID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	_, err := h.tryCreate(ctx, deploymentID, h.imageRequest())
	require.Error(t, err, "a row belonging to another app must not be answered as this caller's")
}

// TestCreateBlocks covers the refusals. Each is a successful invocation
// carrying a reason rather than a failure: the GitHub webhook sends Create
// one-way, so a workspace that will never be eligible must not leave a failed
// invocation behind every push.
func TestCreateBlocks(t *testing.T) {
	t.Run("workspace has no Compute plan", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})
		h.clearComputePlan(t, ctx)

		resp := h.create(t, context.Background(), uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_BLOCKED, resp.GetOutcome())
		require.Equal(t, hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_NO_COMPUTE_PLAN, resp.GetBlockedReason())
		require.Zero(t, h.countDeployments(t, ctx), "a blocked create must write nothing")
	})

	// Observe mode is how the plan gate rolls out: it reports what it would have
	// stopped without stopping anything.
	t.Run("no plan passes while the gate only observes", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{observeGateOnly: true})
		h.clearComputePlan(t, ctx)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, resp.GetOutcome())
	})

	t.Run("workspace is spend suspended", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{observeGateOnly: true})
		h.suspendSpend(t, ctx)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_BLOCKED, resp.GetOutcome())
		require.Equal(t, hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_SPEND_SUSPENDED, resp.GetBlockedReason(),
			"the spend cap blocks even when plan enforcement only observes")
	})

	t.Run("target no longer exists", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		req := h.imageRequest()
		req.Environment = uid.New(uid.EnvironmentPrefix)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_TARGET_NOT_FOUND, resp.GetBlockedReason(),
			"an environment deleted mid-create is a block, not an error: no retry brings it back")
	})

	// Refused rather than quietly redeployed as the current image, which would
	// build a different artifact than the caller asked for.
	t.Run("git source without a repository connection", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.gitRequest())
		require.Equal(t, hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_NO_REPO_CONNECTION, resp.GetBlockedReason())
		require.Zero(t, h.countDeployments(t, ctx))
	})

	t.Run("source deployment has nothing to build from", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		// Neither a commit nor an image: a build that never produced an artifact.
		source := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   h.workspaceID,
			ProjectID:     h.projectID,
			AppID:         h.appID,
			EnvironmentID: h.environmentID,
			Status:        mysqltype.DeploymentsStatusFailed,
		})

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.existingRequest(source.ID, false))
		require.Equal(t, hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_NO_SOURCE_IMAGE, resp.GetBlockedReason())
	})

	// An operator rebuild sets the guardrail; force clears it. Resurrecting a
	// deployment someone has already shipped past is almost never intended.
	t.Run("a newer deployment already exists", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		// Both rows are image deployments with no branch. Siblings are matched
		// with MySQL's NULL-safe equal, so two non-git deployments still see
		// each other; plain equality would return UNKNOWN and skip the guard.
		source := h.imageDeployment(t, ctx, time.Now().Add(-time.Hour).UnixMilli())
		h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   h.workspaceID,
			ProjectID:     h.projectID,
			AppID:         h.appID,
			EnvironmentID: h.environmentID,
			Status:        mysqltype.DeploymentsStatusReady,
			CreatedAt:     time.Now().UnixMilli(),
		})

		guarded := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.existingRequest(source.ID, true))
		require.Equal(t, hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_NEWER_DEPLOYMENT_EXISTS, guarded.GetBlockedReason())

		forced := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.existingRequest(source.ID, false))
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, forced.GetOutcome(),
			"clearing the guardrail is how an operator forces the rebuild through")
	})
}

// TestCreatePushReceivedAtBecomesCreatedAt pins what keeps push order. The
// webhook sends one Create per app asynchronously, so they land in any order,
// and created_at is what sibling dedup and the supersede check compare.
func TestCreatePushReceivedAtBecomesCreatedAt(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})

	pushedAt := time.Now().Add(-90 * time.Second).UnixMilli()
	req := h.imageRequest()
	req.PushReceivedAt = pushedAt

	deploymentID := uid.New(uid.DeploymentPrefix)
	h.create(t, ctx, deploymentID, req)

	require.Equal(t, pushedAt, h.deployment(t, ctx, deploymentID).CreatedAt)
}

// TestCreateFromExistingDeployment covers the source arm behind operator
// rebuilds and redeploys.
func TestCreateFromExistingDeployment(t *testing.T) {
	t.Run("rebuilds the commit while the repository is connected", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})
		h.connectRepo(t, ctx)

		source := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:               uid.New(uid.DeploymentPrefix),
			WorkspaceID:      h.workspaceID,
			ProjectID:        h.projectID,
			AppID:            h.appID,
			EnvironmentID:    h.environmentID,
			Status:           mysqltype.DeploymentsStatusReady,
			GitCommitSha:     sql.NullString{Valid: true, String: fixtureCommitSHA},
			GitBranch:        sql.NullString{Valid: true, String: "main"},
			GitCommitMessage: sql.NullString{Valid: true, String: fixtureCommitMessage},
		})

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		sent := h.awaitDeploy(t, deploymentID)
		git, ok := sent.GetSource().(*hydrav1.DeployRequest_Git)
		require.True(t, ok, "a connected repository rebuilds from git")
		require.Equal(t, fixtureCommitSHA, git.Git.GetCommitSha())
		require.Equal(t, fixtureRepo, git.Git.GetRepository())

		require.Equal(t, fixtureCommitSHA, h.deployment(t, ctx, deploymentID).GitCommitSha.String,
			"the new row records the commit it reproduces")
	})

	t.Run("reuses the image when no repository is connected", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		source := h.imageDeployment(t, ctx, 0)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		sent := h.awaitDeploy(t, deploymentID)
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_DockerImage)
		require.True(t, ok, "without a connection there is no commit to rebuild")
		require.Equal(t, fixtureImage, image.DockerImage.GetImage())
	})

	// An operator rebuild is audited as deployment.rebuild naming both
	// deployments, so a customer's feed shows which deployment replaced which.
	// The trigger is what marks it as Unkey's doing.
	t.Run("an operator rebuild is audited as a rebuild", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		source := h.imageDeployment(t, ctx, 0)

		req := h.existingRequest(source.ID, false)
		req.Trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_UNKEY
		req.TriggerReason = "image lost from the registry"
		req.Actor = &ctrlv1.ActorInfo{
			Id:        "unkey-ops",
			Name:      "Unkey Ops",
			Type:      ctrlv1.ActorType_ACTOR_TYPE_SYSTEM,
			RemoteIp:  "",
			UserAgent: "",
			Meta:      map[string]string{"reason": "image lost from the registry"},
		}

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, req)

		require.Equal(t, 1, h.countAudits(t, ctx, auditlog.DeploymentRebuildEvent, deploymentID))
		require.Zero(t, h.countAudits(t, ctx, auditlog.DeploymentCreateEvent, deploymentID),
			"a rebuild records its own event instead of a create")

		payload := h.auditPayload(t, ctx, auditlog.DeploymentRebuildEvent, deploymentID)
		require.Contains(t, payload, "unkey-ops", "the operator actor must survive onto the audit entry")
		require.Contains(t, payload, source.ID, "the audit names the deployment being replaced")
	})
}

// TestCreateSkipWritesRowWithoutBuilding pins the skipped row: it records a
// commit that was seen and deliberately not built, so conditions that refuse a
// real deployment, such as a missing repository connection, must not refuse it.
func TestCreateSkipWritesRowWithoutBuilding(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})
	h.seedEnvVar(t, ctx, "SECRET_TOKEN", "KEBAP")

	// A git source with no repository connection, which would block a deploy.
	deploymentID := uid.New(uid.DeploymentPrefix)
	req := h.gitRequest()
	req.Decision = hydrav1.CreateDecision_CREATE_DECISION_SKIP

	resp := h.create(t, ctx, deploymentID, req)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, resp.GetOutcome(),
		"a skip has no source to resolve, so a missing connection cannot block it")

	row := h.deployment(t, ctx, deploymentID)
	require.Equal(t, mysqltype.DeploymentsStatusSkipped, row.Status)
	require.Equal(t, fixtureCommitSHA, row.GitCommitSha.String, "the row records the commit it skipped")

	// A row that never builds has no business holding the environment's secrets.
	require.NotContains(t, string(row.EncryptedEnvironmentVariables), "KEBAP")

	require.Zero(t, h.countAudits(t, ctx, auditlog.DeploymentCreateEvent, deploymentID),
		"nothing happened that a customer needs to see in their audit feed")
	h.requireNoDeploy(t, deploymentID)
}

// TestCreateAwaitApprovalDoesNotBuild pins the fork-PR row: only
// AuthorizeDeployment may start it building, so until then no external code
// runs.
func TestCreateAwaitApprovalDoesNotBuild(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})
	h.connectRepo(t, ctx)
	h.seedEnvVar(t, ctx, "SECRET_TOKEN", "KEBAP")

	deploymentID := uid.New(uid.DeploymentPrefix)
	req := h.gitRequest()
	req.Decision = hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL

	h.create(t, ctx, deploymentID, req)

	row := h.deployment(t, ctx, deploymentID)
	require.Equal(t, mysqltype.DeploymentsStatusAwaitingApproval, row.Status)

	// It will build once approved, so unlike a skip it carries its secrets.
	require.Contains(t, string(row.EncryptedEnvironmentVariables), "KEBAP")

	require.Equal(t, 1, h.countAudits(t, ctx, auditlog.DeploymentCreateEvent, deploymentID),
		"a deployment waiting for approval is still a deployment the customer created")
	h.requireNoDeploy(t, deploymentID)
}

// createHarness is one MySQL database and one Restate server hosting the real
// Create next to a stand-in for Deploy.
type createHarness struct {
	database db.Database
	seeder   *seed.Seeder
	client   *restateingress.Client
	deploys  *createDeployRecorder

	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

type createHarnessOptions struct {
	// observeGateOnly runs the plan gate in warn-only mode, which is how it
	// rolls out. The spend cap is enforced either way.
	observeGateOnly bool
}

func newCreateHarness(t *testing.T, ctx context.Context, opts createHarnessOptions) *createHarness {
	t.Helper()

	database, fixture := newDeployFixture(t, ctx)

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
		EnforceDeployGate:               !opts.observeGateOnly,
	})
	require.NoError(t, err)

	recorder := &createDeployRecorder{
		Workflow: workflow,
		requests: make(map[string]*hydrav1.DeployRequest),
	}

	// GitHubStatusService has to be bound: Create sends it the
	// awaiting-authorization commit status, and Restate retries a call to an
	// unregistered service indefinitely, which would hang that test.
	cfg := containers.Restate(t,
		hydrav1.NewDeployServiceServer(recorder),
		hydrav1.NewGitHubStatusServiceServer(githubstatus.New(githubstatus.Config{
			GitHub:                          githubclient.NewNoop(),
			DB:                              database,
			AllowUnauthenticatedDeployments: true,
		})),
	)

	h := &createHarness{
		database:      database,
		seeder:        fixture.seeder,
		client:        cfg.IngressClient,
		deploys:       recorder,
		workspaceID:   fixture.workspaceID,
		projectID:     fixture.projectID,
		appID:         fixture.appID,
		environmentID: fixture.environmentID,
	}
	h.grantComputePlan(t, ctx)
	return h
}

// createDeployRecorder is the real workflow with Deploy replaced. Create sends
// Deploy to itself, so the real Create has to run while building does not.
type createDeployRecorder struct {
	*deploy.Workflow
	mu       sync.Mutex
	requests map[string]*hydrav1.DeployRequest
}

func (r *createDeployRecorder) Deploy(_ restate.ObjectContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests[req.GetDeploymentId()] = req
	return &hydrav1.DeployResponse{}, nil
}

func (r *createDeployRecorder) get(deploymentID string) *hydrav1.DeployRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests[deploymentID]
}

func (r *createDeployRecorder) forget(deploymentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.requests, deploymentID)
}

func (h *createHarness) create(t *testing.T, ctx context.Context, deploymentID string, req *hydrav1.DeployCreateRequest) *hydrav1.DeployCreateResponse {
	t.Helper()
	resp, err := h.tryCreate(ctx, deploymentID, req)
	require.NoError(t, err)
	return resp
}

func (h *createHarness) tryCreate(ctx context.Context, deploymentID string, req *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	return hydrav1.NewDeployServiceIngressClient(h.client, deploymentID).Create().Request(ctx, req)
}

// imageRequest is a create that needs no GitHub and no repository connection,
// so it isolates whatever a test is actually about.
func (h *createHarness) imageRequest() *hydrav1.DeployCreateRequest {
	return &hydrav1.DeployCreateRequest{
		ProjectId:   h.projectID,
		AppId:       h.appID,
		Environment: h.environmentID,
		Source: &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: fixtureImage},
		},
		Command:        nil,
		Decision:       hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		PushReceivedAt: 0,
		Trigger:        ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API,
		TriggeredBy:    "root_KEBAP",
		TriggerReason:  "",
		Actor: &ctrlv1.ActorInfo{
			Id:        "root_KEBAP",
			Name:      "KEBAP key",
			Type:      ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY,
			RemoteIp:  "",
			UserAgent: "",
			Meta:      nil,
		},
	}
}

// gitRequest carries a complete commit, so the worker has nothing to fetch from
// GitHub and these tests never depend on it answering.
func (h *createHarness) gitRequest() *hydrav1.DeployCreateRequest {
	req := h.imageRequest()
	req.Source = &hydrav1.DeployCreateRequest_Git{
		Git: &hydrav1.CreateGitSource{
			Commit: &ctrlv1.GitCommitInfo{
				CommitSha:       fixtureCommitSHA,
				Branch:          "main",
				CommitMessage:   fixtureCommitMessage,
				AuthorHandle:    "contributor",
				AuthorAvatarUrl: "",
				Timestamp:       time.Now().UnixMilli(),
				ForkRepository:  "",
			},
			PrNumber: 0,
		},
	}
	return req
}

func (h *createHarness) existingRequest(sourceID string, requireNoNewer bool) *hydrav1.DeployCreateRequest {
	req := h.imageRequest()
	req.Source = &hydrav1.DeployCreateRequest_ExistingDeployment{
		ExistingDeployment: &hydrav1.CreateExistingDeploymentSource{
			DeploymentId:   sourceID,
			RequireNoNewer: requireNoNewer,
		},
	}
	return req
}

// imageDeployment is a deployment that produced an image and nothing else, which
// is what a redeploy falls back to when there is no commit to rebuild.
func (h *createHarness) imageDeployment(t *testing.T, ctx context.Context, createdAt int64) db.Deployment {
	t.Helper()
	row := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   h.workspaceID,
		ProjectID:     h.projectID,
		AppID:         h.appID,
		EnvironmentID: h.environmentID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     createdAt,
	})
	require.NoError(t, h.database.UpdateDeploymentImage(ctx, db.UpdateDeploymentImageParams{
		Image:     sql.NullString{Valid: true, String: fixtureImage},
		UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:        row.ID,
	}))
	return row
}

func (h *createHarness) awaitDeploy(t *testing.T, deploymentID string) *hydrav1.DeployRequest {
	t.Helper()
	require.Eventually(t, func() bool {
		return h.deploys.get(deploymentID) != nil
	}, 15*time.Second, 100*time.Millisecond, "Create must dispatch Deploy for deployment %s", deploymentID)
	return h.deploys.get(deploymentID)
}

// requireNoDeploy proves a build never started. The window has to be long enough
// for a Send to have arrived, or the assertion passes for the wrong reason.
func (h *createHarness) requireNoDeploy(t *testing.T, deploymentID string) {
	t.Helper()
	require.Never(t, func() bool {
		return h.deploys.get(deploymentID) != nil
	}, 3*time.Second, 200*time.Millisecond, "deployment %s must not build", deploymentID)
}

func (h *createHarness) deployment(t *testing.T, ctx context.Context, deploymentID string) db.Deployment {
	t.Helper()
	row, err := h.database.FindDeploymentById(ctx, deploymentID)
	require.NoError(t, err)
	return row
}

func (h *createHarness) countDeployments(t *testing.T, ctx context.Context) int {
	t.Helper()
	var count int
	require.NoError(t, h.database.RO().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM deployments WHERE app_id = ?", h.appID,
	).Scan(&count))
	return count
}

// queuedStep returns the queued step's ended_at, or nil while it is still open.
func (h *createHarness) queuedStep(t *testing.T, ctx context.Context, deploymentID string) *int64 {
	t.Helper()
	var endedAt *int64
	require.NoError(t, h.database.RO().QueryRowContext(ctx,
		"SELECT ended_at FROM deployment_steps WHERE deployment_id = ? AND step = 'queued'", deploymentID,
	).Scan(&endedAt))
	return endedAt
}

// countAudits counts audit events of one kind naming one deployment. Audit logs
// land in the clickhouse outbox.
func (h *createHarness) countAudits(t *testing.T, ctx context.Context, event auditlog.AuditLogEvent, deploymentID string) int {
	t.Helper()
	rows, err := h.database.ListClickhouseOutboxByWorkspace(ctx, h.workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		payload := string(row.Payload)
		if strings.Contains(payload, string(event)) && strings.Contains(payload, deploymentID) {
			count++
		}
	}
	return count
}

func (h *createHarness) auditPayload(t *testing.T, ctx context.Context, event auditlog.AuditLogEvent, deploymentID string) string {
	t.Helper()
	rows, err := h.database.ListClickhouseOutboxByWorkspace(ctx, h.workspaceID)
	require.NoError(t, err)

	for _, row := range rows {
		payload := string(row.Payload)
		if strings.Contains(payload, string(event)) && strings.Contains(payload, deploymentID) {
			return payload
		}
	}
	t.Fatalf("no %s audit entry for deployment %s", event, deploymentID)
	return ""
}

func (h *createHarness) grantComputePlan(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE workspace_billing SET plan_override = ? WHERE workspace_id = ?", "starter", h.workspaceID)
	require.NoError(t, err)
}

func (h *createHarness) clearComputePlan(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE workspace_billing SET plan = NULL, plan_override = NULL WHERE workspace_id = ?", h.workspaceID)
	require.NoError(t, err)
}

func (h *createHarness) suspendSpend(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE workspace_billing SET spend_suspended = 1 WHERE workspace_id = ?", h.workspaceID)
	require.NoError(t, err)
}

func (h *createHarness) connectRepo(t *testing.T, ctx context.Context) {
	t.Helper()
	require.NoError(t, h.database.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
		WorkspaceID:        h.workspaceID,
		ProjectID:          h.projectID,
		AppID:              h.appID,
		InstallationID:     12345,
		RepositoryID:       67890,
		RepositoryFullName: fixtureRepo,
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{Valid: false},
	}))
}

func (h *createHarness) seedEnvVar(t *testing.T, ctx context.Context, key, value string) {
	t.Helper()
	require.NoError(t, h.database.InsertAppEnvironmentVariable(ctx, db.InsertAppEnvironmentVariableParams{
		ID:            uid.New(uid.EnvironmentVariablePrefix),
		WorkspaceID:   h.workspaceID,
		AppID:         h.appID,
		EnvironmentID: h.environmentID,
		EnvKey:        key,
		Value:         value,
		CreatedAt:     time.Now().UnixMilli(),
	}))
}

// newApp adds a second app with its own environment, for the cases that need a
// target this caller is not deploying to.
func (h *createHarness) newApp(t *testing.T, ctx context.Context) deployFixture {
	t.Helper()
	app := h.seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   h.workspaceID,
		ProjectID:     h.projectID,
		Name:          "KEBAP",
		Slug:          deploySlug(uid.AppPrefix),
		DefaultBranch: "main",
	})
	environment := h.seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: h.workspaceID,
		ProjectID:   h.projectID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})
	return deployFixture{
		seeder:        h.seeder,
		workspaceID:   h.workspaceID,
		projectID:     h.projectID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}
