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
	image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
	require.True(t, ok, "an image source must reach Deploy as an image")
	require.Equal(t, fixtureImage, image.OciImage.GetImage())

	require.Eventually(t, func() bool {
		current := h.deployment(t, ctx, deploymentID)
		return current.InvocationID.Valid && current.InvocationID.String != ""
	}, 15*time.Second, 100*time.Millisecond, "the create must record the Deploy invocation id")

	require.Equal(t, 1, h.countAudits(t, ctx, auditlog.DeploymentCreateEvent, deploymentID))
}

// TestCreateRejections covers the refusals. Each is a successful invocation
// carrying a reason rather than a failure: the GitHub webhook sends Create
// one-way, so a workspace that will never be eligible must not leave a failed
// invocation behind every push.
func TestCreateRejections(t *testing.T) {
	t.Run("workspace has no Compute plan", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})
		h.clearComputePlan(t, ctx)

		resp := h.create(t, context.Background(), uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_REJECTED, resp.GetOutcome())
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_COMPUTE_PLAN, resp.GetRejectionReason())
		require.Zero(t, h.countDeployments(t, ctx), "a rejected create must write nothing")
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
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_REJECTED, resp.GetOutcome())
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SPEND_SUSPENDED, resp.GetRejectionReason(),
			"the spend cap blocks even when plan enforcement only observes")
	})

	t.Run("target no longer exists", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		req := h.imageRequest()
		req.Environment = uid.New(uid.EnvironmentPrefix)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_TARGET_NOT_FOUND, resp.GetRejectionReason(),
			"an environment deleted mid-create is a rejection, not an error: no retry brings it back")
	})

	// Refused rather than quietly redeployed as the current image, which would
	// build a different artifact than the caller asked for.
	t.Run("git source without a repository connection", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.gitRequest())
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_REPO_CONNECTION, resp.GetRejectionReason())
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
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_SOURCE_IMAGE, resp.GetRejectionReason())
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
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NEWER_DEPLOYMENT_EXISTS, guarded.GetRejectionReason())

		forced := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.existingRequest(source.ID, false))
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, forced.GetOutcome(),
			"clearing the guardrail is how an operator forces the rebuild through")
	})

	t.Run("image reference is not valid", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		req := h.imageRequest()
		req.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: "ghcr.io/unkey/KEBAP:v1"},
		}

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_INVALID_IMAGE, resp.GetRejectionReason())
		require.Zero(t, h.countDeployments(t, ctx), "a reference no build could pull must not reach a row")
	})

	// The environment's own settings, not the request: a deployment written
	// against them could only ever reach FAILED.
	t.Run("environment has nowhere to schedule", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})
		h.clearRegions(t, ctx)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_ENVIRONMENT_NOT_DEPLOYABLE, resp.GetRejectionReason())
		require.Zero(t, h.countDeployments(t, ctx))
	})

	t.Run("environment runtime settings are out of bounds", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})
		h.setPort(t, ctx, 0)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_ENVIRONMENT_NOT_DEPLOYABLE, resp.GetRejectionReason())
	})

	t.Run("no source named and the app never deployed", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx, createHarnessOptions{})

		req := h.imageRequest()
		req.Source = nil

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_SOURCE_IMAGE, resp.GetRejectionReason())
	})
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
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok, "without a connection there is no commit to rebuild")
		require.Equal(t, fixtureImage, image.OciImage.GetImage())
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

// TestDeployTargetScoping covers every way the (project, app, environment)
// triple can fail to line up. All of them miss: the query decides the triple as
// a whole, so a caller never learns from the result that an app it cannot reach
// exists, and Create turns every miss into one TARGET_NOT_FOUND rejected.
func TestDeployTargetScoping(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})

	otherProject := h.seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: h.workspaceID,
		Name:        "KEBAP",
		Slug:        deploySlug(uid.ProjectPrefix),
	})
	otherApp := h.seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   h.workspaceID,
		ProjectID:     otherProject.ID,
		Name:          "KEBAP",
		Slug:          deploySlug(uid.AppPrefix),
		DefaultBranch: "main",
	})
	foreignEnv := h.seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: h.workspaceID,
		ProjectID:   otherProject.ID,
		AppID:       otherApp.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})

	// An environment on the right app carrying no settings rows, reachable only
	// by inserting directly: the seeder always writes settings alongside.
	bareEnvID := uid.New(uid.EnvironmentPrefix)
	require.NoError(t, h.database.InsertEnvironment(ctx, db.InsertEnvironmentParams{
		ID:          bareEnvID,
		WorkspaceID: h.workspaceID,
		ProjectID:   h.projectID,
		AppID:       h.appID,
		Slug:        "bare",
		Description: "",
		Kind:        mysqltype.EnvironmentKindPreview,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   sql.NullInt64{Valid: false, Int64: 0},
	}))

	// An environment hanging off the right app but stamped with another project,
	// which only a bug or a half-finished move produces.
	require.NoError(t, h.database.InsertEnvironment(ctx, db.InsertEnvironmentParams{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: h.workspaceID,
		ProjectID:   otherProject.ID,
		AppID:       h.appID,
		Slug:        "stray",
		Description: "",
		Kind:        mysqltype.EnvironmentKindPreview,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   sql.NullInt64{Valid: false, Int64: 0},
	}))

	misses := []struct {
		name      string
		projectID string
		appID     string
		env       string
	}{
		{name: "unknown project", projectID: uid.New(uid.ProjectPrefix), appID: h.appID, env: "production"},
		{name: "unknown app", projectID: h.projectID, appID: uid.New(uid.AppPrefix), env: "production"},
		{name: "app in another project", projectID: h.projectID, appID: otherApp.ID, env: "production"},
		{name: "unknown environment slug", projectID: h.projectID, appID: h.appID, env: "staging"},
		{name: "empty environment", projectID: h.projectID, appID: h.appID, env: ""},
		{name: "environment in another project", projectID: h.projectID, appID: h.appID, env: "stray"},
		{name: "environment without settings", projectID: h.projectID, appID: h.appID, env: "bare"},
		{name: "unknown environment id", projectID: h.projectID, appID: h.appID, env: uid.New(uid.EnvironmentPrefix)},
		{name: "environment id under another app", projectID: h.projectID, appID: h.appID, env: foreignEnv.ID},
		{name: "environment id without settings", projectID: h.projectID, appID: h.appID, env: bareEnvID},
	}

	for _, tt := range misses {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.database.FindDeployTarget(ctx, db.FindDeployTargetParams{
				ProjectID:   tt.projectID,
				AppID:       tt.appID,
				Environment: tt.env,
			})
			require.True(t, db.IsNotFound(err), "want a miss, got %v", err)
		})
	}

	// The settings a create copies onto the row. A join that dropped one of the
	// settings tables would still pass every miss above.
	target, err := h.database.FindDeployTarget(ctx, db.FindDeployTargetParams{
		ProjectID:   h.projectID,
		AppID:       h.appID,
		Environment: "production",
	})
	require.NoError(t, err)
	require.Equal(t, h.environmentID, target.EnvironmentID)
	require.Equal(t, "main", target.DefaultBranch)
	require.Equal(t, "Dockerfile", target.Dockerfile.String)
	require.Equal(t, ".", target.DockerContext)
	require.Equal(t, int32(8080), target.Port)
	require.Equal(t, int32(250), target.CpuMillicores)
	require.Equal(t, int32(256), target.MemoryMib)

	// A rebuild names the environment by id instead, which must land on the same
	// row: the two lookups differ only in that predicate.
	byID, err := h.database.FindDeployTarget(ctx, db.FindDeployTargetParams{
		ProjectID:   h.projectID,
		AppID:       h.appID,
		Environment: h.environmentID,
	})
	require.NoError(t, err)
	require.Equal(t, target, byID)
}

// TestCreateWithoutSource covers the arm a caller uses when it knows only that
// it wants this app shipped again. It splits on the repository connection the
// same way the legacy RPC did: a connected app deploys the head of its default
// branch, and only an app without one redeploys what it runs now.
func TestCreateWithoutSource(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})

	current := h.imageDeployment(t, ctx, time.Now().Add(-time.Hour).UnixMilli())
	h.setCurrentDeployment(t, ctx, current.ID)

	req := h.imageRequest()
	req.Source = nil

	deploymentID := uid.New(uid.DeploymentPrefix)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
		h.create(t, ctx, deploymentID, req).GetOutcome())

	sent := h.awaitDeploy(t, deploymentID)
	image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
	require.True(t, ok, "an app with no repository connection redeploys its image")
	require.Equal(t, fixtureImage, image.OciImage.GetImage())
}

// TestCreateWithoutSourceOnConnectedAppResolvesGit is the other half, and the
// rejection is the assertion. The harness has no GitHub, so resolving the head
// of the default branch necessarily fails there — which is exactly what proves
// the create took the git path. Redeploying the current deployment instead
// would have succeeded with an image source and turned "deploy my app" into
// "redeploy what is already running", which the legacy RPC pointedly did not do.
func TestCreateWithoutSourceOnConnectedAppResolvesGit(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})
	h.connectRepo(t, ctx)

	current := h.imageDeployment(t, ctx, time.Now().Add(-time.Hour).UnixMilli())
	h.setCurrentDeployment(t, ctx, current.ID)

	// The inference below only holds while the current deployment carries no
	// commit: with one, the reverted path would reach GitHub too and fail the
	// same way, and this test would pass for the wrong reason.
	require.False(t, current.GitCommitSha.Valid, "the current deployment must have no commit to rebuild")

	req := h.imageRequest()
	req.Source = nil

	deploymentID := uid.New(uid.DeploymentPrefix)
	resp := h.create(t, ctx, deploymentID, req)
	require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_COMMIT_NOT_RESOLVED,
		resp.GetRejectionReason(),
		"a connected app must resolve the branch head, not fall back to the current image")

	// The current deployment carries an image, so the old behavior would have
	// written a row and dispatched it.
	require.Equal(t, 1, h.countDeployments(t, ctx), "only the seeded current deployment")
	h.requireNoDeploy(t, deploymentID)
}

// TestCreateFromForeignDeploymentIsRejected pins the ownership check. The id is
// caller-supplied and looked up by primary key alone, so without it a request
// could rebuild another app's deployment into its own and run an image it has no
// right to pull.
func TestCreateFromForeignDeploymentIsRejected(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})

	other := h.newApp(t, ctx)
	foreign := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   h.workspaceID,
		ProjectID:     h.projectID,
		AppID:         other.appID,
		EnvironmentID: other.environmentID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	require.NoError(t, h.database.UpdateDeploymentImage(ctx, db.UpdateDeploymentImageParams{
		Image:     sql.NullString{Valid: true, String: "ghcr.io/someone-else/private:v1"},
		UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:        foreign.ID,
	}))

	deploymentID := uid.New(uid.DeploymentPrefix)
	resp := h.create(t, ctx, deploymentID, h.existingRequest(foreign.ID, false))
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_REJECTED, resp.GetOutcome())
	require.Equal(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SOURCE_DEPLOYMENT_NOT_FOUND,
		resp.GetRejectionReason(),
		"a deployment under another app must answer exactly like one that does not exist")
	require.Zero(t, h.countDeployments(t, ctx), "nothing may be written from a foreign source")
	h.requireNoDeploy(t, deploymentID)
}

// TestInsertDeploymentToleratesACommittedRow covers the recovery that keeps a
// lost commit acknowledgement from wedging a create. TxRetry re-runs the whole
// transaction whenever the failure looks transient, and a commit whose ack never
// arrived looks exactly like that; the second attempt then hits a duplicate key
// on a row that is already correct.
func TestInsertDeploymentToleratesACommittedRow(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})

	deploymentID := uid.New(uid.DeploymentPrefix)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
		h.create(t, ctx, deploymentID, h.imageRequest()).GetOutcome())

	first := h.deployment(t, ctx, deploymentID)

	// A second create on the same key is what a re-executed insert stage does to
	// the database: the row is already there, and reporting that as a failure
	// would burn every retry on an error no attempt can clear.
	resp, err := h.tryCreate(ctx, deploymentID, h.imageRequest())
	require.NoError(t, err, "a committed row must not fail the create")
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, resp.GetOutcome())

	after := h.deployment(t, ctx, deploymentID)
	require.Equal(t, first.CreatedAt, after.CreatedAt, "the committed row wins")
	require.Equal(t, 1, h.countDeployments(t, ctx), "no second row")
}

// TestCreateSkipIgnoresEnvironmentDeployability keeps the record of a push that
// was deliberately not built. Refusing the skip would leave the push with no
// record at all, which is what the reason on the row exists to prevent.
func TestCreateSkipIgnoresEnvironmentDeployability(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx, createHarnessOptions{})
	h.clearRegions(t, ctx)

	req := h.gitRequest()
	req.Decision = hydrav1.CreateDecision_CREATE_DECISION_SKIP
	req.TriggerReason = "Watch paths did not match any changed files."

	deploymentID := uid.New(uid.DeploymentPrefix)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
		h.create(t, ctx, deploymentID, req).GetOutcome())

	row := h.deployment(t, ctx, deploymentID)
	require.Equal(t, mysqltype.DeploymentsStatusSkipped, row.Status)
	require.Equal(t, "Watch paths did not match any changed files.", row.TriggerReason.String)
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

	cfg := containers.Restate(t, hydrav1.NewDeployServiceServer(recorder))

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
		Decision:      hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Trigger:       ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API,
		TriggeredBy:   "root_KEBAP",
		TriggerReason: "",
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

func (h *createHarness) clearRegions(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"DELETE FROM app_regional_settings WHERE app_id = ?", h.appID)
	require.NoError(t, err)
}

func (h *createHarness) setPort(t *testing.T, ctx context.Context, port int32) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE app_runtime_settings SET port = ? WHERE app_id = ?", port, h.appID)
	require.NoError(t, err)
}

func (h *createHarness) setCurrentDeployment(t *testing.T, ctx context.Context, deploymentID string) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE apps SET current_deployment_id = ? WHERE id = ?", deploymentID, h.appID)
	require.NoError(t, err)
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
