package api

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/idempotency"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/deployment"
)

type mockDeployService struct {
	hydrav1.UnimplementedDeployServiceServer
	requests chan *hydrav1.DeployRequest
}

func (m *mockDeployService) Deploy(ctx restate.ObjectContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	m.requests <- req
	return &hydrav1.DeployResponse{}, nil
}

func TestDeployment_Create_TriggersWorkflow(t *testing.T) {
	requests := make(chan *hydrav1.DeployRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: requests})},
	})

	ctx := harness.RequestContext()
	workspaceID := harness.Seed.Resources.UserWorkspace.ID
	project := harness.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      workspaceID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})

	envID := uid.New("env")

	app := harness.CreateAppWithSettings(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	}, envID)

	environment := harness.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               envID,
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Kind:             mysqltype.EnvironmentKindProduction,
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	// Seed a schedulable region and regional settings so the environment passes
	// the deployability gate in CreateDeployment, which requires at least one
	// schedulable region.
	region := harness.Seed.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     "us-east-1",
		Platform: "test",
	})
	require.NoError(t, harness.DB.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		RegionID:      region.ID,
		Replicas:      1,
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     sql.NullInt64{Valid: false},
	}))

	client := ctrlv1connect.NewDeployServiceClient(harness.ConnectClient(), harness.CtrlURL, harness.ConnectOptions()...)
	resp, err := client.CreateDeployment(ctx, connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		ProjectId:       project.ID,
		AppId:           app.ID,
		EnvironmentSlug: environment.Slug,
		DockerImage:     "nginx:latest",
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetDeploymentId())
	require.Equal(t, ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING, resp.Msg.GetStatus())

	select {
	case req := <-requests:
		require.Equal(t, resp.Msg.GetDeploymentId(), req.GetDeploymentId())
		dockerImage, ok := req.GetSource().(*hydrav1.DeployRequest_DockerImage)
		require.True(t, ok, "expected DockerImage source")
		require.Equal(t, "nginx:latest", dockerImage.DockerImage.GetImage())
	case <-time.After(10 * time.Second):
		t.Fatal("expected deployment workflow invocation")
	}

	deployment, err := harness.DB.FindDeploymentById(ctx, resp.Msg.GetDeploymentId())
	require.NoError(t, err)
	require.Equal(t, project.ID, deployment.ProjectID)
	require.Equal(t, mysqltype.DeploymentsStatusPending, deployment.Status)
}

func TestDeployment_Create_IdempotencyKey(t *testing.T) {
	requests := make(chan *hydrav1.DeployRequest, 16)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: requests})},
	})

	ctx := harness.RequestContext()
	workspaceID := harness.Seed.Resources.UserWorkspace.ID
	target := seedDeployTarget(ctx, t, harness, workspaceID)

	client := ctrlv1connect.NewDeployServiceClient(harness.ConnectClient(), harness.CtrlURL, harness.ConnectOptions()...)

	create := func(tgt deployTarget, key string) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
		// nolint: exhaustruct // only the fields the dedup path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       tgt.project.ID,
			AppId:           tgt.app.ID,
			EnvironmentSlug: tgt.environment.Slug,
			DockerImage:     "nginx:latest",
		}
		if key != "" {
			req.IdempotencyKey = &key
		}
		return client.CreateDeployment(ctx, connect.NewRequest(req))
	}

	// derivedID mirrors the id the service derives for a key, so a test can
	// seed a row the retry will collide with.
	derivedID := func(key string) string {
		return uid.Derived(uid.DeploymentPrefix, workspaceID, key)
	}

	// expectWorkflows asserts exactly n Deploy invocations arrive, then silence.
	expectWorkflows := func(t *testing.T, n int) {
		t.Helper()
		for i := range n {
			select {
			case <-requests:
			case <-time.After(10 * time.Second):
				t.Fatalf("expected %d deployment workflow invocations, got %d", n, i)
			}
		}
		// Short silence window: an unwanted duplicate send is enqueued before
		// the create RPC returns, so it lands within milliseconds of the
		// expected ones. Every subtest pays this wait in full.
		select {
		case req := <-requests:
			t.Fatalf("unexpected extra workflow invocation for deployment %s", req.GetDeploymentId())
		case <-time.After(500 * time.Millisecond):
		}
	}

	t.Run("same key twice returns the original deployment", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, key)
		require.NoError(t, err)
		second, err := create(target, key)
		require.NoError(t, err)

		require.Equal(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		require.False(t, first.Msg.GetReplayed(), "the creating request must not report a replay")
		require.True(t, second.Msg.GetReplayed(), "the retry must report a replay")

		// The requested image is recorded at insert time so a heal can pin
		// the build to it.
		row, err := harness.DB.FindDeploymentById(ctx, first.Msg.GetDeploymentId())
		require.NoError(t, err)
		require.Equal(t, "nginx:latest", row.Image.String, "a docker create must record its image at insert")

		expectWorkflows(t, 1)
	})

	t.Run("a key bound to another app is rejected", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		_, err := create(target, key)
		require.NoError(t, err)

		otherTarget := seedDeployTarget(ctx, t, harness, workspaceID)
		_, err = create(otherTarget, key)
		require.Error(t, err, "a key bound to one app must not answer for another")
		require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))

		var cErr *connect.Error
		require.ErrorAs(t, err, &cErr)
		require.Equal(t, idempotency.ReasonScopeMismatch, cErr.Meta().Get(idempotency.MetaKey))
		expectWorkflows(t, 1)
	})

	t.Run("different keys create different deployments", func(t *testing.T) {
		first, err := create(target, uid.New(uid.TestPrefix))
		require.NoError(t, err)
		second, err := create(target, uid.New(uid.TestPrefix))
		require.NoError(t, err)

		require.NotEqual(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		expectWorkflows(t, 2)
	})

	t.Run("same key in another workspace creates its own deployment", func(t *testing.T) {
		otherWorkspace := harness.Seed.CreateWorkspaceWithLimits(ctx, seed.CreateWorkspaceWithLimitsRequest{
			RequestsPerMonth:       1_000_000,
			LogsRetentionDays:      30,
			AuditLogsRetentionDays: 30,
			Team:                   false,
		})
		otherTarget := seedDeployTarget(ctx, t, harness, otherWorkspace.ID)

		key := uid.New(uid.TestPrefix)
		first, err := create(target, key)
		require.NoError(t, err)
		second, err := create(otherTarget, key)
		require.NoError(t, err)

		require.NotEqual(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		expectWorkflows(t, 2)
	})

	t.Run("concurrent creates with one key produce one deployment", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		const workers = 4
		ids := make([]string, workers)
		replays := make([]bool, workers)
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := range ids {
			wg.Go(func() {
				resp, err := create(target, key)
				errs[i] = err
				if err == nil {
					ids[i] = resp.Msg.GetDeploymentId()
					replays[i] = resp.Msg.GetReplayed()
				}
			})
		}
		wg.Wait()

		created := 0
		for i := range ids {
			require.NoError(t, errs[i])
			require.Equal(t, ids[0], ids[i])
			if !replays[i] {
				created++
			}
		}
		require.Equal(t, 1, created, "exactly one request may report creating the deployment")

		// Only the inserting request audits: a loser that heals the winner's
		// still-settling row must not log a second create.
		require.Equal(t, 1, countCreateAuditLogs(ctx, t, harness, workspaceID, ids[0]))
		expectWorkflows(t, 1)
	})

	t.Run("replay writes no second audit log", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, key)
		require.NoError(t, err)
		deploymentID := first.Msg.GetDeploymentId()
		require.Equal(t, 1, countCreateAuditLogs(ctx, t, harness, workspaceID, deploymentID))

		_, err = create(target, key)
		require.NoError(t, err)
		require.Equal(t, 1, countCreateAuditLogs(ctx, t, harness, workspaceID, deploymentID))
		expectWorkflows(t, 1)
	})

	// A create that failed before starting a workflow keeps its id, and nothing
	// reclaims it. The retry has to say so instead of returning a deployment
	// that will never run, so the caller knows to send a new key.
	t.Run("a key bound to a create that never started is spent", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)

		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            deploymentID,
			WorkspaceID:   workspaceID,
			ProjectID:     target.project.ID,
			AppID:         target.app.ID,
			EnvironmentID: target.environment.ID,
			Status:        mysqltype.DeploymentsStatusFailed,
			CreatedAt:     time.Now().UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
		})

		_, err := create(target, key)
		require.Error(t, err)
		require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))

		// The reason must survive the wire so the API can tell a spent key
		// apart from any other AlreadyExists.
		var cErr *connect.Error
		require.ErrorAs(t, err, &cErr)
		require.Equal(t, idempotency.ReasonKeySpent, cErr.Meta().Get(idempotency.MetaKey))

		// The dead row is left exactly as it was.
		kept, err := harness.DB.FindDeploymentById(ctx, deploymentID)
		require.NoError(t, err)
		require.Equal(t, mysqltype.DeploymentsStatusFailed, kept.Status)
		expectWorkflows(t, 0)

		// The caller rotates and gets a deployment. This is the whole recovery
		// path: a spent key costs one wasted call, not the ability to deploy.
		rotated, err := create(target, uid.New(uid.TestPrefix))
		require.NoError(t, err)
		require.NotEqual(t, deploymentID, rotated.Msg.GetDeploymentId())
		expectWorkflows(t, 1)
	})

	// Failed is not the only way a deployment can die before its workflow ran:
	// sibling dedup skips it, or an operator stops it. All of them spend the
	// key the same way, because replaying a deployment that never ran and never
	// will leaves the caller polling forever.
	t.Run("a key bound to a dead deployment that never ran is spent regardless of status", func(t *testing.T) {
		for _, status := range []mysqltype.DeploymentsStatus{
			mysqltype.DeploymentsStatusSkipped,
			mysqltype.DeploymentsStatusStopped,
			mysqltype.DeploymentsStatusSuperseded,
			mysqltype.DeploymentsStatusCancelled,
		} {
			key := uid.New(uid.TestPrefix)

			harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
				ID:            derivedID(key),
				WorkspaceID:   workspaceID,
				ProjectID:     target.project.ID,
				AppID:         target.app.ID,
				EnvironmentID: target.environment.ID,
				Status:        status,
				CreatedAt:     time.Now().UnixMilli(),
				UpdatedAt:     sql.NullInt64{Valid: false},
			})

			_, err := create(target, key)
			require.Error(t, err, "status %s must spend the key", status)
			require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err), "status %s", status)
		}
		expectWorkflows(t, 0)
	})

	// Ready is the one terminal status that cannot spend the key: reaching it
	// proves the workflow ran, so a lost invocation id must replay the
	// success, not tell the caller to rotate.
	t.Run("replay returns a ready deployment whose invocation id was lost", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)

		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            deploymentID,
			WorkspaceID:   workspaceID,
			ProjectID:     target.project.ID,
			AppID:         target.app.ID,
			EnvironmentID: target.environment.ID,
			Status:        mysqltype.DeploymentsStatusReady,
			CreatedAt:     time.Now().UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
		})

		resp, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, deploymentID, resp.Msg.GetDeploymentId())
		require.True(t, resp.Msg.GetReplayed())
		require.Equal(t, ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY, resp.Msg.GetStatus())
		expectWorkflows(t, 0)
	})

	// Awaiting-approval rows have no invocation id by design: their send
	// happens at authorization. A keyed retry must replay them, never heal
	// (which would skip the approval gate) and never spend the key.
	t.Run("replay returns an awaiting-approval deployment without sending its workflow", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)

		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            deploymentID,
			WorkspaceID:   workspaceID,
			ProjectID:     target.project.ID,
			AppID:         target.app.ID,
			EnvironmentID: target.environment.ID,
			Status:        mysqltype.DeploymentsStatusAwaitingApproval,
			CreatedAt:     time.Now().UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
		})

		resp, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, deploymentID, resp.Msg.GetDeploymentId())
		require.True(t, resp.Msg.GetReplayed())
		require.Equal(t, ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_AWAITING_APPROVAL, resp.Msg.GetStatus())
		expectWorkflows(t, 0)
	})

	// The scope check has two arms; the other-app subtest above exercises the
	// app arm, this one isolates the environment arm.
	t.Run("a key bound to another environment of the same app is rejected", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            derivedID(key),
			WorkspaceID:   workspaceID,
			ProjectID:     target.project.ID,
			AppID:         target.app.ID,
			EnvironmentID: uid.New("env"),
			Status:        mysqltype.DeploymentsStatusPending,
			CreatedAt:     time.Now().UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
		})

		_, err := create(target, key)
		require.Error(t, err)
		require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))

		var cErr *connect.Error
		require.ErrorAs(t, err, &cErr)
		require.Equal(t, idempotency.ReasonScopeMismatch, cErr.Meta().Get(idempotency.MetaKey))
		expectWorkflows(t, 0)
	})

	t.Run("replay returns a deployment that failed after its workflow started", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)

		// An invocation id marks a deployment whose workflow ran and failed
		// for real: the key stays bound to it and no new workflow starts.
		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            deploymentID,
			WorkspaceID:   workspaceID,
			ProjectID:     target.project.ID,
			AppID:         target.app.ID,
			EnvironmentID: target.environment.ID,
			Status:        mysqltype.DeploymentsStatusFailed,
			CreatedAt:     time.Now().UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
		})
		require.NoError(t, harness.DB.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
			ID:           deploymentID,
			InvocationID: sql.NullString{Valid: true, String: uid.New("inv")},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))

		resp, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, deploymentID, resp.Msg.GetDeploymentId())
		require.Equal(t, ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_FAILED, resp.Msg.GetStatus(),
			"a replay must answer with the deployment's actual status, not a hardcoded pending")
		expectWorkflows(t, 0)
	})

	t.Run("replay settles before gates and source resolution", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, key)
		require.NoError(t, err)

		// Same key, different body: a git commit on an app without a repo
		// connection fails source resolution, so success proves the replay
		// returned before the gates and source resolution ran.
		// nolint: exhaustruct // only the fields the replay path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       target.project.ID,
			AppId:           target.app.ID,
			EnvironmentSlug: target.environment.Slug,
			GitCommit:       &ctrlv1.GitCommitInfo{CommitSha: "0123456789abcdef0123456789abcdef01234567"},
			IdempotencyKey:  &key,
		}
		second, err := client.CreateDeployment(ctx, connect.NewRequest(req))
		require.NoError(t, err)

		require.Equal(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		expectWorkflows(t, 1)
	})

	t.Run("no key creates a deployment per request", func(t *testing.T) {
		first, err := create(target, "")
		require.NoError(t, err)
		second, err := create(target, "")
		require.NoError(t, err)

		require.NotEqual(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		expectWorkflows(t, 2)
	})

	// A key only binds to a deployment once one exists. That is why the
	// dashboard can keep one key across a corrected resubmit instead of
	// rotating on every edit: a rejected request creates nothing, so the key
	// is still free.
	t.Run("a rejected attempt does not bind the key", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// A git commit on an app with no repo connection fails source
		// resolution, which runs before the insert.
		// nolint: exhaustruct // only the fields the reject path reads matter here
		rejected := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       target.project.ID,
			AppId:           target.app.ID,
			EnvironmentSlug: target.environment.Slug,
			GitCommit:       &ctrlv1.GitCommitInfo{CommitSha: "0123456789abcdef0123456789abcdef01234567"},
			IdempotencyKey:  &key,
		}
		_, err := client.CreateDeployment(ctx, connect.NewRequest(rejected))
		require.Error(t, err)

		_, findErr := harness.DB.FindDeploymentById(ctx, derivedID(key))
		require.Error(t, findErr, "a rejected create must leave no row for the key to bind to")

		// Same key, corrected body: this must deploy, not replay.
		resp, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, derivedID(key), resp.Msg.GetDeploymentId())
		expectWorkflows(t, 1)
	})

	// A create killed between the row insert and the workflow send leaves a
	// pending row with no invocation id. The retry heals it: instead of
	// replaying a deployment that would never run, it sends the workflow and
	// backfills the invocation id.
	t.Run("a create that died before its workflow started is healed", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)

		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            deploymentID,
			WorkspaceID:   workspaceID,
			ProjectID:     target.project.ID,
			AppID:         target.app.ID,
			EnvironmentID: target.environment.ID,
			Status:        mysqltype.DeploymentsStatusPending,
			CreatedAt:     time.Now().Add(-time.Hour).UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
		})

		// Every insert records a source, so give the stuck row the image its
		// create resolved: the heal rebuilds what the id already means.
		require.NoError(t, harness.DB.UpdateDeploymentImage(ctx, db.UpdateDeploymentImageParams{
			ID:        deploymentID,
			Image:     sql.NullString{Valid: true, String: "nginx:KEBAP"},
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))

		resp, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, deploymentID, resp.Msg.GetDeploymentId())
		require.True(t, resp.Msg.GetReplayed(), "a heal answers with an existing deployment")

		expectWorkflows(t, 1)

		healed, err := harness.DB.FindDeploymentById(ctx, deploymentID)
		require.NoError(t, err)
		require.True(t, healed.InvocationID.Valid, "heal must backfill the invocation id")

		// The heal did not insert, so it audits nothing. This row was seeded
		// straight into MySQL and has no audit; a row inserted by the service
		// commits its audit atomically with the insert.
		require.Equal(t, 0, countCreateAuditLogs(ctx, t, harness, workspaceID, deploymentID))
	})

	// A heal must deploy what the stuck row records, not what the retry body
	// resolves to. Between the original attempt and the retry the branch may
	// have moved (or the caller changed the body); building the new commit
	// while the row records the old one makes the deployment metadata lie.
	t.Run("heal builds the commit the stuck row records", func(t *testing.T) {
		gitTarget := seedDeployTarget(ctx, t, harness, workspaceID)
		require.NoError(t, harness.DB.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
			WorkspaceID:        workspaceID,
			ProjectID:          gitTarget.project.ID,
			AppID:              gitTarget.app.ID,
			InstallationID:     12345,
			RepositoryID:       67890,
			RepositoryFullName: "unkeyed/KEBAP",
			CreatedAt:          time.Now().UnixMilli(),
			UpdatedAt:          sql.NullInt64{Valid: false},
		}))

		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)
		recordedSHA := strings.Repeat("a", 40)

		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:               deploymentID,
			WorkspaceID:      workspaceID,
			ProjectID:        gitTarget.project.ID,
			AppID:            gitTarget.app.ID,
			EnvironmentID:    gitTarget.environment.ID,
			Status:           mysqltype.DeploymentsStatusPending,
			CreatedAt:        time.Now().Add(-time.Hour).UnixMilli(),
			UpdatedAt:        sql.NullInt64{Valid: false},
			GitCommitSha:     sql.NullString{Valid: true, String: recordedSHA},
			GitBranch:        sql.NullString{Valid: true, String: "main"},
			GitCommitMessage: sql.NullString{Valid: true, String: "original commit"},
		})

		// The retry carries full commit metadata so ctrl resolves nothing via
		// GitHub, and a different SHA, as if the branch head moved.
		// nolint: exhaustruct // only the fields the heal path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       gitTarget.project.ID,
			AppId:           gitTarget.app.ID,
			EnvironmentSlug: gitTarget.environment.Slug,
			GitCommit: &ctrlv1.GitCommitInfo{
				CommitSha:     strings.Repeat("b", 40),
				Branch:        "main",
				CommitMessage: "branch head moved",
			},
			IdempotencyKey: &key,
		}
		resp, err := client.CreateDeployment(ctx, connect.NewRequest(req))
		require.NoError(t, err)
		require.Equal(t, deploymentID, resp.Msg.GetDeploymentId())

		select {
		case deployReq := <-requests:
			git := deployReq.GetGit()
			require.NotNil(t, git, "a git-recorded row must heal as a git build")
			require.Equal(t, recordedSHA, git.GetCommitSha(),
				"heal must build the commit the row records, not the retry body")
		case <-time.After(10 * time.Second):
			t.Fatal("expected the heal to send a deployment workflow")
		}
		expectWorkflows(t, 0)
	})

	// The image is pinned the same way the commit is: the row records what
	// this deployment id means, so a retry with a different image (a caller
	// with broken key rotation) must not smuggle a new artifact under the
	// original id.
	t.Run("heal builds the image the stuck row records", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)

		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            deploymentID,
			WorkspaceID:   workspaceID,
			ProjectID:     target.project.ID,
			AppID:         target.app.ID,
			EnvironmentID: target.environment.ID,
			Status:        mysqltype.DeploymentsStatusPending,
			CreatedAt:     time.Now().Add(-time.Hour).UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
		})
		require.NoError(t, harness.DB.UpdateDeploymentImage(ctx, db.UpdateDeploymentImageParams{
			ID:        deploymentID,
			Image:     sql.NullString{Valid: true, String: "nginx:original"},
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))

		// nolint: exhaustruct // only the fields the heal path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       target.project.ID,
			AppId:           target.app.ID,
			EnvironmentSlug: target.environment.Slug,
			DockerImage:     "nginx:changed",
			IdempotencyKey:  &key,
		}
		resp, err := client.CreateDeployment(ctx, connect.NewRequest(req))
		require.NoError(t, err)
		require.Equal(t, deploymentID, resp.Msg.GetDeploymentId())

		select {
		case deployReq := <-requests:
			require.Equal(t, "nginx:original", deployReq.GetDockerImage().GetImage(),
				"heal must build the image the row records, not the retry body")
		case <-time.After(10 * time.Second):
			t.Fatal("expected the heal to send a deployment workflow")
		}
		expectWorkflows(t, 0)
	})

	// A pending git row has no image yet (the workflow writes it after the
	// build), so the image pin alone cannot protect it. The recorded commit
	// must keep the row a git build: a retry body that switches to a docker
	// image must not repoint the id at a different artifact.
	t.Run("heal keeps a git row a git build when the retry body switches to an image", func(t *testing.T) {
		gitTarget := seedDeployTarget(ctx, t, harness, workspaceID)
		require.NoError(t, harness.DB.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
			WorkspaceID:        workspaceID,
			ProjectID:          gitTarget.project.ID,
			AppID:              gitTarget.app.ID,
			InstallationID:     12345,
			RepositoryID:       67890,
			RepositoryFullName: "unkeyed/KEBAP",
			CreatedAt:          time.Now().UnixMilli(),
			UpdatedAt:          sql.NullInt64{Valid: false},
		}))

		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)
		recordedSHA := strings.Repeat("c", 40)

		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:               deploymentID,
			WorkspaceID:      workspaceID,
			ProjectID:        gitTarget.project.ID,
			AppID:            gitTarget.app.ID,
			EnvironmentID:    gitTarget.environment.ID,
			Status:           mysqltype.DeploymentsStatusPending,
			CreatedAt:        time.Now().Add(-time.Hour).UnixMilli(),
			UpdatedAt:        sql.NullInt64{Valid: false},
			GitCommitSha:     sql.NullString{Valid: true, String: recordedSHA},
			GitBranch:        sql.NullString{Valid: true, String: "main"},
			GitCommitMessage: sql.NullString{Valid: true, String: "original commit"},
		})

		// nolint: exhaustruct // only the fields the heal path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       gitTarget.project.ID,
			AppId:           gitTarget.app.ID,
			EnvironmentSlug: gitTarget.environment.Slug,
			DockerImage:     "nginx:evil",
			IdempotencyKey:  &key,
		}
		resp, err := client.CreateDeployment(ctx, connect.NewRequest(req))
		require.NoError(t, err)
		require.Equal(t, deploymentID, resp.Msg.GetDeploymentId())

		select {
		case deployReq := <-requests:
			git := deployReq.GetGit()
			require.NotNil(t, git,
				"a git-recorded row must heal as a git build even when the retry body carries an image")
			require.Equal(t, recordedSHA, git.GetCommitSha())
		case <-time.After(10 * time.Second):
			t.Fatal("expected the heal to send a deployment workflow")
		}
		expectWorkflows(t, 0)
	})

	// A git row whose repo connection was deleted between the crash and the
	// retry must refuse, not silently fall back to redeploying the app's
	// current image under the row's commit metadata. And because no retry can
	// ever build it, the refusal must also fail the row so the key spends:
	// otherwise the caller loops on the same key forever with no exit.
	t.Run("heal refuses a git row whose repo connection is gone", func(t *testing.T) {
		goneTarget := seedDeployTarget(ctx, t, harness, workspaceID)

		// Give the app a current deployment with an image, so the silent
		// fallback arm is reachable and the refusal is what stops it.
		currentID := uid.New(uid.DeploymentPrefix)
		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            currentID,
			WorkspaceID:   workspaceID,
			ProjectID:     goneTarget.project.ID,
			AppID:         goneTarget.app.ID,
			EnvironmentID: goneTarget.environment.ID,
			Status:        mysqltype.DeploymentsStatusReady,
			CreatedAt:     time.Now().Add(-2 * time.Hour).UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
		})
		require.NoError(t, harness.DB.UpdateDeploymentImage(ctx, db.UpdateDeploymentImageParams{
			ID:        currentID,
			Image:     sql.NullString{Valid: true, String: "nginx:current"},
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))
		require.NoError(t, harness.DB.SetAppCurrentDeployment(ctx, db.SetAppCurrentDeploymentParams{
			DeploymentID: sql.NullString{Valid: true, String: currentID},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			AppID:        goneTarget.app.ID,
		}))

		key := uid.New(uid.TestPrefix)
		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:               derivedID(key),
			WorkspaceID:      workspaceID,
			ProjectID:        goneTarget.project.ID,
			AppID:            goneTarget.app.ID,
			EnvironmentID:    goneTarget.environment.ID,
			Status:           mysqltype.DeploymentsStatusPending,
			CreatedAt:        time.Now().Add(-time.Hour).UnixMilli(),
			UpdatedAt:        sql.NullInt64{Valid: false},
			GitCommitSha:     sql.NullString{Valid: true, String: strings.Repeat("d", 40)},
			GitBranch:        sql.NullString{Valid: true, String: "main"},
			GitCommitMessage: sql.NullString{Valid: true, String: "original commit"},
		})

		_, err := create(goneTarget, key)
		require.Error(t, err,
			"a git row without its repo connection must refuse, not redeploy another artifact")
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		expectWorkflows(t, 0)

		row, findErr := harness.DB.FindDeploymentById(ctx, derivedID(key))
		require.NoError(t, findErr)
		require.Equal(t, mysqltype.DeploymentsStatusPending, row.Status,
			"the refusal leaves the row healable, so reconnecting the repo makes the same key work")

		// This reaches API callers as a 412, and clients drop the key on a 4xx,
		// so the caller sends a new key instead of looping on this one.
		_, retryErr := create(goneTarget, key)
		require.Error(t, retryErr)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(retryErr))
		expectWorkflows(t, 0)
	})

	// A failed pre-check read must fail the request, not fall through to the
	// full create path (gates, source resolution, insert) against a database
	// that is already erroring.
	t.Run("a failed key lookup fails the create", func(t *testing.T) {
		findTarget := seedDeployTarget(ctx, t, harness, workspaceID)

		auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: harness.DB})
		require.NoError(t, err)
		svc := deployment.New(deployment.Config{
			Database:                        failingFindDB{harness.DB},
			Restate:                         restateingress.NewClient(harness.IngressURL),
			RestateAdmin:                    nil,
			GitHub:                          nil,
			Auditlogs:                       auditlogSvc,
			AllowUnauthenticatedDeployments: false,
			Bearer:                          harness.AuthToken,
			EnforceDeployGate:               false,
		})

		key := uid.New(uid.TestPrefix)
		// nolint: exhaustruct // only the fields the create path reads matter here
		req := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
			ProjectId:       findTarget.project.ID,
			AppId:           findTarget.app.ID,
			EnvironmentSlug: findTarget.environment.Slug,
			DockerImage:     "nginx:latest",
			IdempotencyKey:  &key,
		})
		req.Header().Set("Authorization", "Bearer "+harness.AuthToken)

		_, createErr := svc.CreateDeployment(ctx, req)
		require.Error(t, createErr, "a broken key lookup must fail the request")
		require.Equal(t, connect.CodeInternal, connect.CodeOf(createErr))
		expectWorkflows(t, 0)
	})

	// The API edge trims the header, but the derivation must defend itself:
	// any direct ctrl caller passing " k " and "k" must land on one deployment.
	t.Run("a key is trimmed before it derives the id", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, "  "+key+" ")
		require.NoError(t, err)
		second, err := create(target, key)
		require.NoError(t, err)

		require.Equal(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId(),
			"whitespace around a key must not derive a different deployment")
		require.True(t, second.Msg.GetReplayed())
		expectWorkflows(t, 1)
	})

	// A heal acts for the original create, so sibling dedup must use the stuck
	// row's age. Using the retry's time would let an old stuck row, healed
	// after someone deployed a newer commit, supersede the newer deployment.
	t.Run("heal does not supersede newer siblings", func(t *testing.T) {
		healTarget := seedDeployTarget(ctx, t, harness, workspaceID)
		require.NoError(t, harness.DB.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
			WorkspaceID:        workspaceID,
			ProjectID:          healTarget.project.ID,
			AppID:              healTarget.app.ID,
			InstallationID:     12345,
			RepositoryID:       67890,
			RepositoryFullName: "unkeyed/KEBAP",
			CreatedAt:          time.Now().UnixMilli(),
			UpdatedAt:          sql.NullInt64{Valid: false},
		}))

		key := uid.New(uid.TestPrefix)
		stuckID := derivedID(key)
		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:               stuckID,
			WorkspaceID:      workspaceID,
			ProjectID:        healTarget.project.ID,
			AppID:            healTarget.app.ID,
			EnvironmentID:    healTarget.environment.ID,
			Status:           mysqltype.DeploymentsStatusPending,
			CreatedAt:        time.Now().Add(-time.Hour).UnixMilli(),
			UpdatedAt:        sql.NullInt64{Valid: false},
			GitCommitSha:     sql.NullString{Valid: true, String: strings.Repeat("a", 40)},
			GitBranch:        sql.NullString{Valid: true, String: "main"},
			GitCommitMessage: sql.NullString{Valid: true, String: "stuck commit"},
		})

		// A newer pending deployment on the same branch, created long after
		// the stuck row.
		newerID := uid.New(uid.DeploymentPrefix)
		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:               newerID,
			WorkspaceID:      workspaceID,
			ProjectID:        healTarget.project.ID,
			AppID:            healTarget.app.ID,
			EnvironmentID:    healTarget.environment.ID,
			Status:           mysqltype.DeploymentsStatusPending,
			CreatedAt:        time.Now().UnixMilli(),
			UpdatedAt:        sql.NullInt64{Valid: false},
			GitCommitSha:     sql.NullString{Valid: true, String: strings.Repeat("b", 40)},
			GitBranch:        sql.NullString{Valid: true, String: "main"},
			GitCommitMessage: sql.NullString{Valid: true, String: "newer commit"},
		})

		// nolint: exhaustruct // only the fields the heal path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       healTarget.project.ID,
			AppId:           healTarget.app.ID,
			EnvironmentSlug: healTarget.environment.Slug,
			GitCommit: &ctrlv1.GitCommitInfo{
				CommitSha:     strings.Repeat("a", 40),
				Branch:        "main",
				CommitMessage: "stuck commit",
			},
			IdempotencyKey: &key,
		}
		resp, err := client.CreateDeployment(ctx, connect.NewRequest(req))
		require.NoError(t, err)
		require.Equal(t, stuckID, resp.Msg.GetDeploymentId())
		expectWorkflows(t, 1)

		newer, err := harness.DB.FindDeploymentById(ctx, newerID)
		require.NoError(t, err)
		require.Equal(t, mysqltype.DeploymentsStatusPending, newer.Status,
			"healing an old stuck row must not supersede a newer deployment")
	})

	// A docker create can carry git attribution (image plus branch, no SHA).
	// The heal must dedup siblings with the branch the row records, not the
	// retry body's: a body without git metadata would silently skip sibling
	// dedup that the original create would have run.
	t.Run("heal dedups with the branch the stuck row records", func(t *testing.T) {
		branchTarget := seedDeployTarget(ctx, t, harness, workspaceID)

		key := uid.New(uid.TestPrefix)
		stuckID := derivedID(key)

		// An older queued sibling on the row's branch, created before the
		// stuck row, so the heal's dedup should supersede it.
		siblingID := uid.New(uid.DeploymentPrefix)
		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:               siblingID,
			WorkspaceID:      workspaceID,
			ProjectID:        branchTarget.project.ID,
			AppID:            branchTarget.app.ID,
			EnvironmentID:    branchTarget.environment.ID,
			Status:           mysqltype.DeploymentsStatusPending,
			CreatedAt:        time.Now().Add(-2 * time.Hour).UnixMilli(),
			UpdatedAt:        sql.NullInt64{Valid: false},
			GitCommitSha:     sql.NullString{Valid: true, String: strings.Repeat("e", 40)},
			GitBranch:        sql.NullString{Valid: true, String: "main"},
			GitCommitMessage: sql.NullString{Valid: true, String: "older commit"},
		})

		// The stuck row: a docker build (image pinned at insert) attributed to
		// a branch, but with no SHA.
		harness.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            stuckID,
			WorkspaceID:   workspaceID,
			ProjectID:     branchTarget.project.ID,
			AppID:         branchTarget.app.ID,
			EnvironmentID: branchTarget.environment.ID,
			Status:        mysqltype.DeploymentsStatusPending,
			CreatedAt:     time.Now().Add(-time.Hour).UnixMilli(),
			UpdatedAt:     sql.NullInt64{Valid: false},
			GitBranch:     sql.NullString{Valid: true, String: "main"},
		})
		require.NoError(t, harness.DB.UpdateDeploymentImage(ctx, db.UpdateDeploymentImageParams{
			ID:        stuckID,
			Image:     sql.NullString{Valid: true, String: "nginx:pinned"},
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))

		// The retry carries no git metadata at all.
		// nolint: exhaustruct // only the fields the heal path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       branchTarget.project.ID,
			AppId:           branchTarget.app.ID,
			EnvironmentSlug: branchTarget.environment.Slug,
			DockerImage:     "nginx:changed",
			IdempotencyKey:  &key,
		}
		resp, err := client.CreateDeployment(ctx, connect.NewRequest(req))
		require.NoError(t, err)
		require.Equal(t, stuckID, resp.Msg.GetDeploymentId())
		expectWorkflows(t, 1)

		sibling, err := harness.DB.FindDeploymentById(ctx, siblingID)
		require.NoError(t, err)
		require.Equal(t, mysqltype.DeploymentsStatusSuperseded, sibling.Status,
			"heal must supersede older siblings on the branch the row records")
	})

	// The other cause of a pending row without an invocation id: the workflow
	// send succeeded but persisting the invocation id failed, so the workflow
	// is already running. The heal re-sends with the deployment id as the
	// Restate idempotency key, so Restate collapses the second send instead of
	// running the build twice.
	t.Run("healing a deployment whose workflow already runs does not double-build", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		deploymentID := derivedID(key)

		first, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, deploymentID, first.Msg.GetDeploymentId())
		expectWorkflows(t, 1)

		// Simulate the lost write: the workflow ran, but the row never
		// recorded its invocation id.
		require.NoError(t, harness.DB.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
			ID:           deploymentID,
			InvocationID: sql.NullString{Valid: false, String: ""},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))

		second, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, deploymentID, second.Msg.GetDeploymentId())
		require.True(t, second.Msg.GetReplayed(), "a heal answers with an existing deployment")

		// The first request audited its insert; the heal adds nothing.
		require.Equal(t, 1, countCreateAuditLogs(ctx, t, harness, workspaceID, deploymentID))

		expectWorkflows(t, 0)

		healed, err := harness.DB.FindDeploymentById(ctx, deploymentID)
		require.NoError(t, err)
		require.True(t, healed.InvocationID.Valid, "heal must recover the invocation id from Restate")
	})

	// A deployment without its deployment.create audit is invisible to the
	// customer's audit trail. The insert and the audit must commit or fail
	// together, so an audit outage cannot leave an unaudited deployment behind.
	t.Run("a create whose audit cannot be written creates nothing", func(t *testing.T) {
		auditTarget := seedDeployTarget(ctx, t, harness, workspaceID)

		svc := deployment.New(deployment.Config{
			Database:                        harness.DB,
			Restate:                         restateingress.NewClient(harness.IngressURL),
			RestateAdmin:                    nil,
			GitHub:                          nil,
			Auditlogs:                       failingAuditlogs{},
			AllowUnauthenticatedDeployments: false,
			Bearer:                          harness.AuthToken,
			EnforceDeployGate:               false,
		})

		key := uid.New(uid.TestPrefix)
		// nolint: exhaustruct // only the fields the create path reads matter here
		req := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
			ProjectId:       auditTarget.project.ID,
			AppId:           auditTarget.app.ID,
			EnvironmentSlug: auditTarget.environment.Slug,
			DockerImage:     "nginx:latest",
			IdempotencyKey:  &key,
		})
		req.Header().Set("Authorization", "Bearer "+harness.AuthToken)

		_, err := svc.CreateDeployment(ctx, req)
		require.Error(t, err, "a create that cannot record its audit must fail")

		_, findErr := harness.DB.FindDeploymentById(ctx, derivedID(key))
		require.Error(t, findErr, "the insert must roll back with the failed audit")

		expectWorkflows(t, 0)
	})
}

// failingAuditlogs simulates an audit outbox outage so tests can prove the
// deployment insert and its audit commit atomically.
type failingAuditlogs struct{}

func (failingAuditlogs) Insert(context.Context, db.DBTX, []auditlog.AuditLog) error {
	return errors.New("KEBAP audit outage")
}

// failingFindDB fails every deployment lookup so tests can prove a broken
// pre-check read fails the create instead of falling through to the insert.
type failingFindDB struct {
	db.Database
}

func (failingFindDB) FindDeploymentById(context.Context, string) (db.Deployment, error) {
	// nolint: exhaustruct // zero row is never read alongside the error
	return db.Deployment{}, errors.New("KEBAP read outage")
}

type deployTarget struct {
	project     db.Project
	app         db.App
	environment db.Environment
}

// seedDeployTarget creates a project, app, environment, and schedulable
// regional settings so CreateDeployment passes the deployability gate.
func seedDeployTarget(ctx context.Context, t *testing.T, harness *webhookHarness, workspaceID string) deployTarget {
	t.Helper()

	project := harness.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      workspaceID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})

	envID := uid.New("env")
	app := harness.CreateAppWithSettings(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	}, envID)

	environment := harness.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               envID,
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Kind:             mysqltype.EnvironmentKindProduction,
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	region := harness.Seed.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     uid.New("rgn"),
		Platform: "test",
	})
	require.NoError(t, harness.DB.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		RegionID:      region.ID,
		Replicas:      1,
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     sql.NullInt64{Valid: false},
	}))

	return deployTarget{project: project, app: app, environment: environment}
}

// countCreateAuditLogs counts deployment.create outbox events that target the
// given deployment. Audit logs land in the clickhouse_outbox table.
func countCreateAuditLogs(ctx context.Context, t *testing.T, harness *webhookHarness, workspaceID, deploymentID string) int {
	t.Helper()

	rows, err := harness.DB.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		payload := string(row.Payload)
		if strings.Contains(payload, string(auditlog.DeploymentCreateEvent)) && strings.Contains(payload, deploymentID) {
			count++
		}
	}
	return count
}
