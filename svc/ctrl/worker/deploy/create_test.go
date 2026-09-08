package deploy_test

import (
	"context"
	"database/sql"
	"fmt"
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
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
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
	h := newCreateHarness(t, ctx)

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
// A CLI deploy pushes an image but says which commit it built. The row keeps
// that so the dashboard can show it and dedup can key on the branch.
func TestCreateImageRecordsCommit(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)

	req := h.imageRequest()
	req.Source = &hydrav1.DeployCreateRequest_Image{
		Image: &hydrav1.CreateImageSource{
			Image: fixtureImage,
			Commit: &ctrlv1.GitCommitInfo{
				Branch:        "release",
				CommitSha:     fixtureCommitSHA,
				CommitMessage: "ship KEBAP",
				AuthorHandle:  "kebap",
			},
		},
	}

	deploymentID := uid.New(uid.DeploymentPrefix)
	resp := h.create(t, ctx, deploymentID, req)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, resp.GetOutcome())

	sent := h.awaitDeploy(t, deploymentID)
	require.Equal(t, fixtureImage, sent.GetOciImage().GetImage(), "an image with a commit is still deployed, not built")

	row := h.deployment(t, ctx, deploymentID)
	require.Equal(t, "release", row.GitBranch.String)
	require.Equal(t, fixtureCommitSHA, row.GitCommitSha.String)
	require.Equal(t, "ship KEBAP", row.GitCommitMessage.String)
	require.Equal(t, "kebap", row.GitCommitAuthorHandle.String)
}

func TestCreateRejections(t *testing.T) {
	t.Run("workspace has no Compute plan", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.clearComputePlan(t, ctx)

		resp := h.create(t, context.Background(), uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_COMPUTE_PLAN, resp.GetOutcome())
		require.Zero(t, h.countDeployments(t, ctx), "a rejected create must write nothing")
	})

	t.Run("workspace is spend suspended", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.suspendSpend(t, ctx)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_SPEND_SUSPENDED, resp.GetOutcome())
	})

	t.Run("target no longer exists", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		req := h.imageRequest()
		req.EnvironmentId = uid.New(uid.EnvironmentPrefix)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_TARGET_NOT_FOUND, resp.GetOutcome(),
			"an environment deleted mid-create is a rejection, not an error: no retry brings it back")
	})

	// Refused rather than quietly redeployed as the current image, which would
	// run something other than what the caller asked for.
	t.Run("git source without a repository connection", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.gitRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_REPO_CONNECTION, resp.GetOutcome())
		require.Zero(t, h.countDeployments(t, ctx))
	})

	t.Run("source deployment has nothing to build from", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		// Neither a commit nor an image: a build that never produced anything.
		source := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   h.workspaceID,
			ProjectID:     h.projectID,
			AppID:         h.appID,
			EnvironmentID: h.environmentID,
			Status:        mysqltype.DeploymentsStatusFailed,
		})

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.existingRequest(source.ID, false))
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE_IMAGE, resp.GetOutcome())
	})

	// An operator rebuild sets the guardrail; force clears it. Resurrecting a
	// deployment someone has already shipped past is almost never intended.
	t.Run("a newer deployment already exists", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

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
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NEWER_DEPLOYMENT_EXISTS, guarded.GetOutcome())
		require.Equal(t, "a newer active deployment exists for this app and environment", guarded.GetDetail())

		forced := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.existingRequest(source.ID, false))
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, forced.GetOutcome(),
			"clearing the guardrail is how an operator forces the rebuild through")
	})

	// Deploy resolves a tag to a digest and refuses an implicit one, so an
	// untagged reference would be written and then fail its first step.
	t.Run("image reference has no tag", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		req := h.imageRequest()
		req.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: "ghcr.io/unkey/kebap"},
		}

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_INVALID_IMAGE, resp.GetOutcome())
		require.Zero(t, h.countDeployments(t, ctx))
	})

	t.Run("image reference is not valid", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		req := h.imageRequest()
		req.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: "ghcr.io/unkey/KEBAP:v1"},
		}

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_INVALID_IMAGE, resp.GetOutcome())
		require.Contains(t, resp.GetDetail(), "ghcr.io/unkey/KEBAP:v1")
		require.Zero(t, h.countDeployments(t, ctx), "a reference no build could pull must not reach a row")
	})

	// The environment's own settings, not the request: a deployment written
	// against them could only ever reach FAILED.
	t.Run("environment has nowhere to schedule", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.clearRegions(t, ctx)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_ENVIRONMENT_NOT_DEPLOYABLE, resp.GetOutcome())
		require.Equal(t, deployfail.MsgNoSchedulableRegions, resp.GetDetail())
		require.Zero(t, h.countDeployments(t, ctx))
	})

	// Every violation is reported at once, so an operator fixing three settings
	// does not need three deploys to discover them.
	t.Run("environment runtime settings are out of bounds", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.setRuntimeBounds(t, ctx, 0, 0, 0)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_ENVIRONMENT_NOT_DEPLOYABLE, resp.GetOutcome())
		require.Contains(t, resp.GetDetail(), deployfail.MsgPortOutOfRange)
		require.Contains(t, resp.GetDetail(), deployfail.MsgCPUTooLow)
		require.Contains(t, resp.GetDetail(), deployfail.MsgMemoryTooLow)
		require.NotContains(t, resp.GetDetail(), deployfail.MsgNoSchedulableRegions,
			"the fixture has a schedulable region, so naming regions would misdirect the fix")
	})

	t.Run("no source named and the app never deployed", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		req := h.imageRequest()
		req.Source = nil

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE, resp.GetOutcome())
	})
}

// TestCreateFromExistingDeployment covers the source arm behind operator
// rebuilds and redeploys.
func TestCreateFromExistingDeployment(t *testing.T) {
	t.Run("rebuilds the commit while the repository is connected", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
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
		h := newCreateHarness(t, ctx)

		source := h.imageDeployment(t, ctx, 0)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		sent := h.awaitDeploy(t, deploymentID)
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok, "without a connection there is no commit to rebuild")
		require.Equal(t, fixtureImage, image.OciImage.GetImage())
	})

	// A user redeploying asked for what that deployment runs. The repository is
	// gone so its commit cannot be rebuilt, and the legacy API answered this with
	// the recorded image rather than a refusal.
	t.Run("a user redeploy falls back to the image once the repository is gone", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		source := h.commitDeployment(t, ctx)
		h.setDeploymentImages(t, ctx, source.ID, db.DeploymentsSourceGit, fixtureImage)

		deploymentID := uid.New(uid.DeploymentPrefix)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
			h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false)).GetOutcome())

		sent := h.awaitDeploy(t, deploymentID)
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok, "with no repository there is nothing to rebuild")
		require.Equal(t, fixtureImage, image.OciImage.GetImage())
	})

	// An operator rebuild is asking for the commit specifically, so reusing the
	// image would rebuild nothing and quietly ship the same code again.
	t.Run("an operator rebuild is refused once the repository is gone", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		source := h.commitDeployment(t, ctx)
		h.setDeploymentImages(t, ctx, source.ID, db.DeploymentsSourceGit, fixtureImage)

		req := h.existingRequest(source.ID, false)
		req.Trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_UNKEY

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE_COMMIT, resp.GetOutcome())
		require.Equal(t, 1, h.countDeployments(t, ctx), "only the seeded source")
	})

	// A CLI deploy records the commit it was built from locally, but what runs
	// is the image. Rebuilding the commit would run our build instead.
	t.Run("an image deployment redeploys its image even with a commit and a connection", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.connectRepo(t, ctx)

		source := h.commitDeployment(t, ctx)
		h.setDeploymentImages(t, ctx, source.ID, db.DeploymentsSourceOci, fixtureImage)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		sent := h.awaitDeploy(t, deploymentID)
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok, "the row's source decides, not the connection")
		require.Equal(t, fixtureImage, image.OciImage.GetImage())
	})

	// Deploy pins the digest it ran into image_resolved. A rebuild reproduces
	// that, not a tag that may since point elsewhere.
	t.Run("a rebuild takes the resolved digest over the requested tag", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		digest := "ghcr.io/unkey/kebap@sha256:" + strings.Repeat("ab", 32)
		source := h.imageDeployment(t, ctx, 0)
		h.setDeploymentImages(t, ctx, source.ID, db.DeploymentsSourceOci, digest)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		sent := h.awaitDeploy(t, deploymentID)
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok)
		require.Equal(t, digest, image.OciImage.GetImage())
	})

	// Rows from before explicit tags were required hold references like "nginx".
	// Deploy refuses those, so the rebuild has to make the tag explicit.
	t.Run("a historical image gets an explicit tag", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		source := h.imageDeployment(t, ctx, 0)
		h.setDeploymentImages(t, ctx, source.ID, db.DeploymentsSourceUnknown, "nginx")

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		sent := h.awaitDeploy(t, deploymentID)
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok)
		require.Equal(t, "index.docker.io/library/nginx:latest", image.OciImage.GetImage())
	})

	// A deployment that arrived as an image has no commit to rebuild, so a
	// repository connected to the app afterwards must not turn it into a build of
	// that repository's default branch.
	t.Run("reuses the image of an image-origin deployment on a connected app", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.connectRepo(t, ctx)

		source := h.imageDeployment(t, ctx, 0)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		sent := h.awaitDeploy(t, deploymentID)
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok, "an image-origin deployment has no commit to rebuild")
		require.Equal(t, fixtureImage, image.OciImage.GetImage())
	})

	// A fork PR's build reads refs/pull/<n>/head from the base repository, so a
	// rebuild that loses the fork or the PR number would build the base branch
	// instead of the contributor's code.
	t.Run("carries the fork and PR number forward", func(t *testing.T) {
		const forkRepo = "contributor/api"
		const prNumber = int64(42)

		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.connectRepo(t, ctx)

		source := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   h.workspaceID,
			ProjectID:     h.projectID,
			AppID:         h.appID,
			EnvironmentID: h.environmentID,
			Status:        mysqltype.DeploymentsStatusReady,
			GitCommitSha:  sql.NullString{Valid: true, String: fixtureCommitSHA},
			GitBranch:     sql.NullString{Valid: true, String: "feature"},
			// Set so the commit needs no completion from GitHub.
			GitCommitMessage:       sql.NullString{Valid: true, String: fixtureCommitMessage},
			PrNumber:               sql.NullInt64{Valid: true, Int64: prNumber},
			ForkRepositoryFullName: sql.NullString{Valid: true, String: forkRepo},
		})

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		sent := h.awaitDeploy(t, deploymentID)
		git, ok := sent.GetSource().(*hydrav1.DeployRequest_Git)
		require.True(t, ok, "a fork PR rebuild is still a git build")
		require.Equal(t, forkRepo, git.Git.GetForkRepository())
		require.Equal(t, prNumber, git.Git.GetPrNumber())
		require.Equal(t, fixtureRepo, git.Git.GetRepository(), "the base repository is what BuildKit fetches the PR ref from")

		row := h.deployment(t, ctx, deploymentID)
		require.Equal(t, forkRepo, row.ForkRepositoryFullName.String)
		require.Equal(t, prNumber, row.PrNumber.Int64)
	})

	// An operator rebuild is audited as deployment.rebuild naming both
	// deployments, so a customer's feed shows which deployment replaced which.
	// The trigger is what marks it as Unkey's doing.
	t.Run("an operator rebuild is audited as a rebuild", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

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
	h := newCreateHarness(t, ctx)
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
	require.Equal(t, db.DeploymentsSourceGit, row.Source, "a skipped push is still a git row")

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
	h := newCreateHarness(t, ctx)
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
	h := newCreateHarness(t, ctx)

	otherProject := h.seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: h.workspaceID,
		Name:        "KEBAP",
		Slug:        deploySlug(uid.ProjectPrefix),
	})
	otherApp := h.seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: h.workspaceID,
		ProjectID:   otherProject.ID,
		Name:        "KEBAP",
		Slug:        deploySlug(uid.AppPrefix),
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
	strayEnvID := uid.New(uid.EnvironmentPrefix)
	require.NoError(t, h.database.InsertEnvironment(ctx, db.InsertEnvironmentParams{
		ID:          strayEnvID,
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
		name          string
		projectID     string
		appID         string
		environmentID string
	}{
		{name: "unknown project", projectID: uid.New(uid.ProjectPrefix), appID: h.appID, environmentID: h.environmentID},
		{name: "unknown app", projectID: h.projectID, appID: uid.New(uid.AppPrefix), environmentID: h.environmentID},
		{name: "app in another project", projectID: h.projectID, appID: otherApp.ID, environmentID: h.environmentID},
		{name: "empty environment", projectID: h.projectID, appID: h.appID, environmentID: ""},
		{name: "unknown environment", projectID: h.projectID, appID: h.appID, environmentID: uid.New(uid.EnvironmentPrefix)},
		{name: "environment in another project", projectID: h.projectID, appID: h.appID, environmentID: strayEnvID},
		{name: "environment under another app", projectID: h.projectID, appID: h.appID, environmentID: foreignEnv.ID},
		{name: "environment without settings", projectID: h.projectID, appID: h.appID, environmentID: bareEnvID},
	}

	for _, tt := range misses {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.database.FindDeployTarget(ctx, db.FindDeployTargetParams{
				ProjectID:     tt.projectID,
				AppID:         tt.appID,
				EnvironmentID: tt.environmentID,
			})
			require.True(t, db.IsNotFound(err), "want a miss, got %v", err)
		})
	}

	// The settings a create copies onto the row. A join that dropped one of the
	// settings tables would still pass every miss above.
	target, err := h.database.FindDeployTarget(ctx, db.FindDeployTargetParams{
		ProjectID:     h.projectID,
		AppID:         h.appID,
		EnvironmentID: h.environmentID,
	})
	require.NoError(t, err)
	require.Equal(t, h.environmentID, target.EnvironmentID)
	require.Equal(t, "Dockerfile", target.Dockerfile.String)
	require.Equal(t, ".", target.DockerContext.String)
	require.Equal(t, int32(8080), target.Port)
	require.Equal(t, int32(250), target.CpuMillicores)
	require.Equal(t, int32(256), target.MemoryMib)
}

// TestCreateWithoutSource covers the arm a caller uses when it knows only that
// it wants this app shipped again. For an app with no declared source it splits
// on the repository connection the same way the legacy RPC did: connected apps
// build the default branch, others redeploy what they run now.
func TestCreateWithoutSource(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)

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

// TestCreateWithoutSourceReusesTheImageOfAGitDeployment redeploys an app whose
// current deployment was built from git but whose repository is gone. Naming a
// deployment asks to reproduce it, so a git build there must rebuild; naming no
// source asks for what the app runs, which is the image.
func TestCreateWithoutSourceReusesTheImageOfAGitDeployment(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)

	current := h.commitDeployment(t, ctx)
	h.setDeploymentImages(t, ctx, current.ID, db.DeploymentsSourceGit, fixtureImage)
	h.setCurrentDeployment(t, ctx, current.ID)

	req := h.imageRequest()
	req.Source = nil

	deploymentID := uid.New(uid.DeploymentPrefix)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
		h.create(t, ctx, deploymentID, req).GetOutcome())

	sent := h.awaitDeploy(t, deploymentID)
	image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
	require.True(t, ok, "with no repository there is nothing to rebuild from")
	require.Equal(t, fixtureImage, image.OciImage.GetImage())
}

// TestCreateWithoutSourceOnConnectedAppResolvesGit is the other half, and the
// rejection is the assertion. The harness has no GitHub, so resolving the head
// of the default branch necessarily fails there, which is exactly what proves
// the create took the git path. Redeploying the current deployment instead
// would have succeeded with an image source and turned "deploy my app" into
// "redeploy what is already running", which the legacy RPC pointedly did not do.
func TestCreateWithoutSourceOnConnectedAppResolvesGit(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)
	h.connectRepo(t, ctx)

	current := h.imageDeployment(t, ctx, time.Now().Add(-time.Hour).UnixMilli())
	h.setCurrentDeployment(t, ctx, current.ID)

	// This only holds while the current deployment carries no commit: with one,
	// the reverted path would reach GitHub too and fail the same way, and this
	// test would pass for the wrong reason.
	require.False(t, current.GitCommitSha.Valid, "the current deployment must have no commit to rebuild")

	req := h.imageRequest()
	req.Source = nil

	deploymentID := uid.New(uid.DeploymentPrefix)
	resp := h.create(t, ctx, deploymentID, req)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_COMMIT_NOT_RESOLVED,
		resp.GetOutcome(),
		"a connected app must resolve the branch head, not fall back to the current image")
	require.Contains(t, resp.GetDetail(), `branch "main"`)

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
	h := newCreateHarness(t, ctx)

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
		ImageResolved: sql.NullString{Valid: true, String: "ghcr.io/someone-else/private:v1"},
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:            foreign.ID,
	}))

	deploymentID := uid.New(uid.DeploymentPrefix)
	resp := h.create(t, ctx, deploymentID, h.existingRequest(foreign.ID, false))
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_SOURCE_DEPLOYMENT_NOT_FOUND,
		resp.GetOutcome(),
		"a deployment under another app must answer exactly like one that does not exist")
	require.Zero(t, h.countDeployments(t, ctx), "nothing may be written from a foreign source")
	h.requireNoDeploy(t, deploymentID)
}

// TestInsertDeploymentToleratesACommittedRow covers the recovery that keeps a
// lost commit acknowledgement from stalling a create. TxRetry re-runs the whole
// transaction whenever the failure looks transient, and a commit whose ack never
// arrived looks exactly like that; the second attempt then hits a duplicate key
// on a row that is already correct.
func TestInsertDeploymentToleratesACommittedRow(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)

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
	h := newCreateHarness(t, ctx)
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

// TestCreateFollowsAppSource pins the app's declared source. An app created as
// OCI has an image and no build settings; one created as Git has the reverse.
// Guessing from the repository connection alone gets both wrong.
func TestCreateFollowsAppSource(t *testing.T) {
	t.Run("an OCI app deploys its configured image", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.setAppSource(t, ctx, db.AppsSourceTypeOci)
		h.seedOciSource(t, ctx, "ghcr.io/unkey/kebap:v2")
		h.dropBuildSettings(t, ctx)

		req := h.imageRequest()
		req.Source = nil

		deploymentID := uid.New(uid.DeploymentPrefix)
		resp := h.create(t, ctx, deploymentID, req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, resp.GetOutcome(),
			"an OCI app has no build settings row and must not need one")

		sent := h.awaitDeploy(t, deploymentID)
		image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok)
		require.Equal(t, "ghcr.io/unkey/kebap:v2", image.OciImage.GetImage())

		row := h.deployment(t, ctx, deploymentID)
		require.Equal(t, db.DeploymentsSourceOci, row.Source)
		require.Equal(t, "ghcr.io/unkey/kebap:v2", row.ImageRequested.String)
		require.False(t, row.GitCommitSha.Valid, "an image deploy synthesizes no commit")
	})

	t.Run("an OCI app with no image configured", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.setAppSource(t, ctx, db.AppsSourceTypeOci)

		req := h.imageRequest()
		req.Source = nil

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_IMAGE_CONFIGURED, resp.GetOutcome())
		require.Zero(t, h.countDeployments(t, ctx))
	})

	// A connection may linger on an app switched to OCI. The declared source wins.
	t.Run("an OCI app refuses a git commit", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.setAppSource(t, ctx, db.AppsSourceTypeOci)
		h.connectRepo(t, ctx)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.gitRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_REPO_CONNECTION, resp.GetOutcome())
		require.Zero(t, h.countDeployments(t, ctx))
	})

	// Falling back to the current image would turn "deploy my app" into
	// "redeploy what is running" for an app whose repository was disconnected.
	t.Run("a git app without a repository connection is refused", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.setAppSource(t, ctx, db.AppsSourceTypeGit)

		current := h.imageDeployment(t, ctx, time.Now().Add(-time.Hour).UnixMilli())
		h.setCurrentDeployment(t, ctx, current.ID)

		req := h.imageRequest()
		req.Source = nil

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_REPO_CONNECTION, resp.GetOutcome())
		require.Equal(t, 1, h.countDeployments(t, ctx), "only the seeded current deployment")
	})

	t.Run("a git app without build settings is refused", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.setAppSource(t, ctx, db.AppsSourceTypeGit)
		h.connectRepo(t, ctx)
		h.dropBuildSettings(t, ctx)

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.gitRequest())
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_ENVIRONMENT_NOT_DEPLOYABLE, resp.GetOutcome())
		require.Zero(t, h.countDeployments(t, ctx))
	})
}

// TestCreateNormalizesImageReference pins the normalized reference on the row
// and in the Deploy request, so the same image never appears under two spellings.
func TestCreateNormalizesImageReference(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)

	req := h.imageRequest()
	req.Source = &hydrav1.DeployCreateRequest_Image{
		Image: &hydrav1.CreateImageSource{Image: "nginx:1.25"},
	}

	deploymentID := uid.New(uid.DeploymentPrefix)
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, h.create(t, ctx, deploymentID, req).GetOutcome())

	sent := h.awaitDeploy(t, deploymentID)
	image, ok := sent.GetSource().(*hydrav1.DeployRequest_OciImage)
	require.True(t, ok)
	require.Equal(t, "index.docker.io/library/nginx:1.25", image.OciImage.GetImage())
	require.Equal(t, "index.docker.io/library/nginx:1.25", h.deployment(t, ctx, deploymentID).ImageRequested.String)
}

// TestDeployTargetCarriesSourceColumns pins the columns the source decision
// reads. The connection's default branch is the one GitHub reported. The app's
// own column is a placeholder on new apps.
func TestDeployTargetCarriesSourceColumns(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)
	h.setAppSource(t, ctx, db.AppsSourceTypeGit)
	h.connectRepo(t, ctx)
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE github_repo_connections SET default_branch = ? WHERE app_id = ?", "release", h.appID)
	require.NoError(t, err)

	target, err := h.database.FindDeployTarget(ctx, db.FindDeployTargetParams{
		ProjectID:     h.projectID,
		AppID:         h.appID,
		EnvironmentID: h.environmentID,
	})
	require.NoError(t, err)
	require.Equal(t, db.AppsSourceTypeGit, target.SourceType)
	require.Equal(t, "release", target.GithubDefaultBranch.String)
	require.True(t, target.HasBuildSettings)
	require.False(t, target.OciImageReference.Valid)

	h.dropBuildSettings(t, ctx)
	target, err = h.database.FindDeployTarget(ctx, db.FindDeployTargetParams{
		ProjectID:     h.projectID,
		AppID:         h.appID,
		EnvironmentID: h.environmentID,
	})
	require.NoError(t, err, "an app without build settings is still a target")
	require.False(t, target.HasBuildSettings)
}

// TestCreateDedupsOnlyTheBranchTheCallerNamed pins which creates supersede a
// queued sibling. A git build and an explicit image both name their branch, so
// they dedup on it. A rebuild that reuses an existing deployment's image names
// none: the branch on its row is inherited so the dashboard can show what the
// image was built from, and superseding on it would cancel builds the caller
// never spoke about.
func TestCreateDedupsOnlyTheBranchTheCallerNamed(t *testing.T) {
	t.Run("an inherited image supersedes nothing", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		now := time.Now().UnixMilli()

		source := h.imageDeploymentOnBranch(t, ctx, "main", now-60_000)
		sibling := h.queuedSibling(t, ctx, "main", now-30_000)

		deploymentID := uid.New(uid.DeploymentPrefix)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
			h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false)).GetOutcome())
		h.awaitDeploy(t, deploymentID)

		require.Equal(t, "main", h.deployment(t, ctx, deploymentID).GitBranch.String,
			"the row still records the branch the image was built from")
		h.requireNotSuperseded(t, ctx, sibling.ID)
	})

	t.Run("an explicit image supersedes its branch", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		sibling := h.queuedSibling(t, ctx, "main", time.Now().UnixMilli()-30_000)

		req := h.imageRequest()
		req.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{
				Image: fixtureImage,
				Commit: &ctrlv1.GitCommitInfo{
					CommitSha:       fixtureCommitSHA,
					Branch:          "main",
					CommitMessage:   fixtureCommitMessage,
					AuthorHandle:    "",
					AuthorAvatarUrl: "",
					Timestamp:       0,
					ForkRepository:  "",
				},
			},
		}

		deploymentID := uid.New(uid.DeploymentPrefix)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
			h.create(t, ctx, deploymentID, req).GetOutcome())
		h.awaitDeploy(t, deploymentID)
		h.requireSuperseded(t, ctx, sibling.ID)
	})

	t.Run("a git build supersedes its branch", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)
		h.setAppSource(t, ctx, db.AppsSourceTypeGit)
		h.connectRepo(t, ctx)
		sibling := h.queuedSibling(t, ctx, "main", time.Now().UnixMilli()-30_000)

		deploymentID := uid.New(uid.DeploymentPrefix)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
			h.create(t, ctx, deploymentID, h.gitRequest()).GetOutcome())
		h.awaitDeploy(t, deploymentID)
		h.requireSuperseded(t, ctx, sibling.ID)
	})
}

// TestCreateRefusesAnIdOwnedByAnotherRow covers the object key, which the
// caller picks. Every gate runs against the request's target, so a key that
// already names a row must never adopt it: the gates never saw it.
func TestCreateRefusesAnIdOwnedByAnotherRow(t *testing.T) {
	t.Run("a deployment belonging to another app", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		other := h.newApp(t, ctx)
		victim := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   other.workspaceID,
			ProjectID:     other.projectID,
			AppID:         other.appID,
			EnvironmentID: other.environmentID,
			Status:        mysqltype.DeploymentsStatusReady,
			CreatedAt:     time.Now().UnixMilli(),
		})

		_, err := h.tryCreate(ctx, victim.ID, h.imageRequest())
		require.Error(t, err, "an id owned by another app must not be adopted")

		h.requireNoDeploy(t, victim.ID)
		after := h.deployment(t, ctx, victim.ID)
		require.False(t, after.InvocationID.Valid, "the other app's invocation id must be untouched")
		require.Equal(t, other.appID, after.AppID, "the row must still belong to the other app")
		require.Equal(t, 0, h.countDeployments(t, ctx), "no row for this app")
	})

	// The tuple matches here, so only the created_at this create journaled
	// separates the row it wrote from one that was already there.
	t.Run("an awaiting-approval row in the same app", func(t *testing.T) {
		ctx := context.Background()
		h := newCreateHarness(t, ctx)

		blocked := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   h.workspaceID,
			ProjectID:     h.projectID,
			AppID:         h.appID,
			EnvironmentID: h.environmentID,
			Status:        mysqltype.DeploymentsStatusAwaitingApproval,
			CreatedAt:     time.Now().UnixMilli(),
		})

		_, err := h.tryCreate(ctx, blocked.ID, h.imageRequest())
		require.Error(t, err, "a create must not adopt a row waiting for approval")

		h.requireNoDeploy(t, blocked.ID)
		require.Equal(t, mysqltype.DeploymentsStatusAwaitingApproval,
			h.deployment(t, ctx, blocked.ID).Status, "the approval gate must still hold")
	})
}

// TestForeignAndMissingSourceDeploymentsAnswerAlike pins the masking. The id is
// caller-supplied, so a foreign deployment has to answer like a miss, or the
// reason lets a caller probe for deployments it cannot reach.
func TestForeignAndMissingSourceDeploymentsAnswerAlike(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)

	other := h.newApp(t, ctx)
	foreign := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   h.workspaceID,
		ProjectID:     h.projectID,
		AppID:         other.appID,
		EnvironmentID: other.environmentID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     time.Now().UnixMilli(),
	})

	missingID := uid.New(uid.DeploymentPrefix)
	missing := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.existingRequest(missingID, false))
	present := h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.existingRequest(foreign.ID, false))

	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_SOURCE_DEPLOYMENT_NOT_FOUND, missing.GetOutcome())
	require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_SOURCE_DEPLOYMENT_NOT_FOUND, present.GetOutcome())

	// One template for both, so the only thing that differs is the id the caller
	// already knew.
	require.Equal(t, fmt.Sprintf("source deployment %s not found", missingID), missing.GetDetail())
	require.Equal(t, fmt.Sprintf("source deployment %s not found", foreign.ID), present.GetDetail())
}

// TestCreateRefusesOversizedIdentifiers covers the columns a commit message
// cannot be trimmed like: a cut sha or branch names something else, so the
// create is refused instead. The row must never reach MySQL and fail there.
func TestCreateRefusesOversizedIdentifiers(t *testing.T) {
	ctx := context.Background()
	h := newCreateHarness(t, ctx)

	// Valid syntax, 530 characters, and imageref puts no bound on total length.
	t.Run("an image reference wider than its column", func(t *testing.T) {
		req := h.imageRequest()
		req.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{
				Image:  strings.Repeat("a.", 260) + "com/foo:v1",
				Commit: nil,
			},
		}

		resp := h.create(t, ctx, uid.New(uid.DeploymentPrefix), req)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_INVALID_IMAGE, resp.GetOutcome())
		require.Equal(t, 0, h.countDeployments(t, ctx), "no row for a refused image")
	})

	t.Run("a branch wider than its column", func(t *testing.T) {
		req := h.imageRequest()
		req.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{
				Image: fixtureImage,
				Commit: &ctrlv1.GitCommitInfo{
					CommitSha:       fixtureCommitSHA,
					Branch:          strings.Repeat("b", 300),
					CommitMessage:   fixtureCommitMessage,
					AuthorHandle:    "",
					AuthorAvatarUrl: "",
					Timestamp:       0,
					ForkRepository:  "",
				},
			},
		}

		_, err := h.tryCreate(ctx, uid.New(uid.DeploymentPrefix), req)
		require.Error(t, err, "a branch that cannot be stored must not be truncated")
		require.NotContains(t, err.Error(), "Data too long",
			"the create has to refuse the branch itself, not spend its retries losing to MySQL")
		require.Equal(t, 0, h.countDeployments(t, ctx), "no row for a refused branch")
	})

	// A full sha is exactly the column width, so the bound has to admit it.
	t.Run("a full length sha is accepted", func(t *testing.T) {
		sha := strings.Repeat("a", 40)
		req := h.imageRequest()
		req.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{
				Image: fixtureImage,
				Commit: &ctrlv1.GitCommitInfo{
					CommitSha:       sha,
					Branch:          "main",
					CommitMessage:   fixtureCommitMessage,
					AuthorHandle:    "",
					AuthorAvatarUrl: "",
					Timestamp:       0,
					ForkRepository:  "",
				},
			},
		}

		deploymentID := uid.New(uid.DeploymentPrefix)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
			h.create(t, ctx, deploymentID, req).GetOutcome())
		require.Equal(t, sha, h.deployment(t, ctx, deploymentID).GitCommitSha.String)
	})

	// git_branch is a utf8mb4 varchar(256), so the column counts characters. A
	// byte count here would refuse a branch that stores fine.
	t.Run("a multi byte branch is measured in characters", func(t *testing.T) {
		branch := strings.Repeat("ケバブ", 80) // 240 characters, 720 bytes
		req := h.imageRequest()
		req.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{
				Image: fixtureImage,
				Commit: &ctrlv1.GitCommitInfo{
					CommitSha:       fixtureCommitSHA,
					Branch:          branch,
					CommitMessage:   fixtureCommitMessage,
					AuthorHandle:    "",
					AuthorAvatarUrl: "",
					Timestamp:       0,
					ForkRepository:  "",
				},
			},
		}

		deploymentID := uid.New(uid.DeploymentPrefix)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
			h.create(t, ctx, deploymentID, req).GetOutcome())
		require.Equal(t, branch, h.deployment(t, ctx, deploymentID).GitBranch.String)
	})
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

func newCreateHarness(t *testing.T, ctx context.Context) *createHarness {
	t.Helper()
	return newCreateHarnessWithAdmin(t, ctx, nil)
}

func newCreateHarnessWithAdmin(t *testing.T, ctx context.Context, admin *restateadmin.Client) *createHarness {
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
		RestateAdmin:                    admin,
	})
	require.NoError(t, err)

	recorder := &createDeployRecorder{
		Workflow: workflow,
		requests: make(map[string]*hydrav1.DeployRequest),
	}

	cfg := containers.Restate(t, hydrav1.NewDeployWorkflowServer(recorder))

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

func (r *createDeployRecorder) Deploy(_ restate.WorkflowContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
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
	return hydrav1.NewDeployWorkflowIngressClient(h.client, deploymentID).Create().Request(ctx, req)
}

// imageRequest is a create that needs no GitHub and no repository connection,
// so it isolates whatever a test is actually about.
func (h *createHarness) imageRequest() *hydrav1.DeployCreateRequest {
	return &hydrav1.DeployCreateRequest{
		ProjectId:     h.projectID,
		AppId:         h.appID,
		EnvironmentId: h.environmentID,
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

func (h *createHarness) existingRequest(sourceID string, requireLatest bool) *hydrav1.DeployCreateRequest {
	req := h.imageRequest()
	req.Source = &hydrav1.DeployCreateRequest_ExistingDeployment{
		ExistingDeployment: &hydrav1.CreateExistingDeploymentSource{
			DeploymentId:  sourceID,
			RequireLatest: requireLatest,
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
		ImageResolved: sql.NullString{Valid: true, String: fixtureImage},
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:            row.ID,
	}))
	return row
}

// imageDeploymentOnBranch is an image deployment that recorded the branch its
// image was built from, which is what the legacy RPC wrote for an oci_image
// create that carried a commit.
func (h *createHarness) imageDeploymentOnBranch(t *testing.T, ctx context.Context, branch string, createdAt int64) db.Deployment {
	t.Helper()
	row := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:               uid.New(uid.DeploymentPrefix),
		WorkspaceID:      h.workspaceID,
		ProjectID:        h.projectID,
		AppID:            h.appID,
		EnvironmentID:    h.environmentID,
		Status:           mysqltype.DeploymentsStatusReady,
		CreatedAt:        createdAt,
		GitBranch:        sql.NullString{Valid: true, String: branch},
		GitCommitSha:     sql.NullString{Valid: true, String: fixtureCommitSHA},
		GitCommitMessage: sql.NullString{Valid: true, String: fixtureCommitMessage},
	})
	require.NoError(t, h.database.UpdateDeploymentImage(ctx, db.UpdateDeploymentImageParams{
		ImageResolved: sql.NullString{Valid: true, String: fixtureImage},
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:            row.ID,
	}))
	return row
}

// queuedSibling is a deployment still in the build queue on a branch, which is
// the only thing dedup is allowed to supersede.
func (h *createHarness) queuedSibling(t *testing.T, ctx context.Context, branch string, createdAt int64) db.Deployment {
	t.Helper()
	return h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   h.workspaceID,
		ProjectID:     h.projectID,
		AppID:         h.appID,
		EnvironmentID: h.environmentID,
		Status:        mysqltype.DeploymentsStatusPending,
		CreatedAt:     createdAt,
		GitBranch:     sql.NullString{Valid: true, String: branch},
	})
}

func (h *createHarness) requireSuperseded(t *testing.T, ctx context.Context, deploymentID string) {
	t.Helper()
	require.Eventually(t, func() bool {
		return h.deployment(t, ctx, deploymentID).Status == mysqltype.DeploymentsStatusSuperseded
	}, 15*time.Second, 100*time.Millisecond, "deployment %s must be superseded", deploymentID)
}

// requireNotSuperseded has to outlast the dedup step, or it passes for the
// wrong reason: dedup runs after Create has already dispatched Deploy.
func (h *createHarness) requireNotSuperseded(t *testing.T, ctx context.Context, deploymentID string) {
	t.Helper()
	require.Never(t, func() bool {
		return h.deployment(t, ctx, deploymentID).Status == mysqltype.DeploymentsStatusSuperseded
	}, 5*time.Second, 200*time.Millisecond, "deployment %s must not be superseded", deploymentID)
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

func (h *createHarness) setRuntimeBounds(t *testing.T, ctx context.Context, port, cpuMillicores, memoryMib int32) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE app_runtime_settings SET port = ?, cpu_millicores = ?, memory_mib = ? WHERE app_id = ?",
		port, cpuMillicores, memoryMib, h.appID)
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
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: h.workspaceID,
		ProjectID:   h.projectID,
		Name:        "KEBAP",
		Slug:        deploySlug(uid.AppPrefix),
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

// commitDeployment is a row that records the commit it was built from.
func (h *createHarness) commitDeployment(t *testing.T, ctx context.Context) db.Deployment {
	t.Helper()
	return h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
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
}

// setDeploymentImages writes what Create and Deploy record about a row's image,
// so a seeded row can stand in for one that ran. An empty resolved image leaves
// the column NULL, as on rows from before Deploy pinned digests.
func (h *createHarness) setDeploymentImages(
	t *testing.T,
	ctx context.Context,
	deploymentID string,
	source db.DeploymentsSource,
	resolved string,
) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE deployments SET source = ?, image_resolved = ? WHERE id = ?",
		string(source), resolved, deploymentID)
	require.NoError(t, err)
}

func (h *createHarness) setAppSource(t *testing.T, ctx context.Context, sourceType db.AppsSourceType) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE apps SET source_type = ? WHERE id = ?", string(sourceType), h.appID)
	require.NoError(t, err)
}

func (h *createHarness) seedOciSource(t *testing.T, ctx context.Context, image string) {
	t.Helper()
	require.NoError(t, h.database.InsertAppSourceOci(ctx, db.InsertAppSourceOciParams{
		WorkspaceID:    h.workspaceID,
		AppID:          h.appID,
		ImageReference: image,
		CreatedAt:      time.Now().UnixMilli(),
		UpdatedAt:      sql.NullInt64{Valid: false},
	}))
}

// dropBuildSettings makes the fixture look like an app created as OCI, which
// gets no build settings row.
func (h *createHarness) dropBuildSettings(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		"DELETE FROM app_build_settings WHERE app_id = ?", h.appID)
	require.NoError(t, err)
}
