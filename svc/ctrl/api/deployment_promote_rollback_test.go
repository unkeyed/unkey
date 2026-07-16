package api

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// mockLifecycleService captures the Promote/Rollback workflow invocations the
// ctrl handler forwards once validation passes. Validation runs in the real ctrl
// service before this is reached, so a rejected request never arrives here.
type mockLifecycleService struct {
	hydrav1.UnimplementedDeployServiceServer
	promotes  chan *hydrav1.PromoteRequest
	rollbacks chan *hydrav1.RollbackRequest
}

func (m *mockLifecycleService) Promote(_ restate.ObjectContext, req *hydrav1.PromoteRequest) (*hydrav1.PromoteResponse, error) {
	m.promotes <- req
	return &hydrav1.PromoteResponse{}, nil
}

func (m *mockLifecycleService) Rollback(_ restate.ObjectContext, req *hydrav1.RollbackRequest) (*hydrav1.RollbackResponse, error) {
	m.rollbacks <- req
	return &hydrav1.RollbackResponse{}, nil
}

// lifecycleFixture is a ctrl service backed by a real database and Restate, plus
// one project/app with a production and a preview environment. Its helpers seed
// deployments in specific states and drive the promote/rollback RPCs, so tests
// read as a state-to-outcome table.
type lifecycleFixture struct {
	t           *testing.T
	ctx         context.Context
	db          db.Database
	seed        *seed.Seeder
	client      ctrlv1connect.DeployServiceClient
	mock        *mockLifecycleService
	workspaceID string
	projectID   string
	appID       string
	prodEnv     db.Environment
	previewEnv  db.Environment
	now         int64
}

func newLifecycleFixture(t *testing.T) *lifecycleFixture {
	t.Helper()

	mock := &mockLifecycleService{
		promotes:  make(chan *hydrav1.PromoteRequest, 8),
		rollbacks: make(chan *hydrav1.RollbackRequest, 8),
	}
	h := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(mock)},
	})

	ctx := h.RequestContext()
	wsID := h.Seed.Resources.UserWorkspace.ID

	project := h.CreateProject(ctx, seed.CreateProjectRequest{
		ID: uid.New("prj"), WorkspaceID: wsID, Name: "test-project", Slug: uid.New("slug"),
	})
	prodEnvID := uid.New("env")
	app := h.CreateAppWithSettings(ctx, seed.CreateAppRequest{
		ID: uid.New("app"), WorkspaceID: wsID, ProjectID: project.ID, Name: "default", Slug: "default", DefaultBranch: "main",
	}, prodEnvID)
	prodEnv := h.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID: prodEnvID, WorkspaceID: wsID, ProjectID: project.ID, AppID: app.ID, Slug: "production", SentinelConfig: []byte("{}"),
	})
	previewEnv := h.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID: uid.New("env"), WorkspaceID: wsID, ProjectID: project.ID, AppID: app.ID, Slug: "preview", SentinelConfig: []byte("{}"),
	})

	return &lifecycleFixture{
		t:           t,
		ctx:         ctx,
		db:          h.DB,
		seed:        h.Seed,
		client:      ctrlv1connect.NewDeployServiceClient(h.ConnectClient(), h.CtrlURL, h.ConnectOptions()...),
		mock:        mock,
		workspaceID: wsID,
		projectID:   project.ID,
		appID:       app.ID,
		prodEnv:     prodEnv,
		previewEnv:  previewEnv,
		now:         time.Now().UnixMilli(),
	}
}

// deployment seeds a deployment in the given environment with an explicit status
// and desired_state.
func (f *lifecycleFixture) deployment(env db.Environment, status db.DeploymentsStatus, desired db.DeploymentsDesiredState) db.Deployment {
	f.t.Helper()
	d := f.seed.CreateDeployment(f.ctx, seed.CreateDeploymentRequest{
		WorkspaceID: f.workspaceID, ProjectID: f.projectID, AppID: f.appID, EnvironmentID: env.ID, Status: status,
	})
	require.NoError(f.t, f.db.UpdateDeploymentDesiredState(f.ctx, db.UpdateDeploymentDesiredStateParams{
		DesiredState: desired, UpdatedAt: sql.NullInt64{Int64: f.now, Valid: true}, ID: d.ID,
	}))
	return d
}

// setLive points the app's current_deployment_id at deploymentID (empty clears
// it) and sets the rolled-back flag.
func (f *lifecycleFixture) setLive(deploymentID string, rolledBack bool) {
	f.t.Helper()
	require.NoError(f.t, f.db.UpdateAppDeployments(f.ctx, db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: deploymentID, Valid: deploymentID != ""},
		IsRolledBack:        rolledBack,
		UpdatedAt:           sql.NullInt64{Int64: f.now, Valid: true},
		AppID:               f.appID,
	}))
}

func (f *lifecycleFixture) promote(id string) error {
	_, err := f.client.Promote(f.ctx, connect.NewRequest(&ctrlv1.PromoteRequest{TargetDeploymentId: id}))
	return err
}

func (f *lifecycleFixture) rollback(sourceID, targetID string) error {
	_, err := f.client.Rollback(f.ctx, connect.NewRequest(&ctrlv1.RollbackRequest{
		SourceDeploymentId: sourceID, TargetDeploymentId: targetID,
	}))
	return err
}

// requirePromoteWorkflow asserts the handler forwarded a promote for targetID to
// the workflow, proving validation passed and delegation carried the right id.
func (f *lifecycleFixture) requirePromoteWorkflow(targetID string) {
	f.t.Helper()
	select {
	case req := <-f.mock.promotes:
		require.Equal(f.t, targetID, req.GetTargetDeploymentId())
	case <-time.After(10 * time.Second):
		f.t.Fatal("expected promote workflow invocation")
	}
}

func (f *lifecycleFixture) requireRollbackWorkflow(sourceID, targetID string) {
	f.t.Helper()
	select {
	case req := <-f.mock.rollbacks:
		require.Equal(f.t, sourceID, req.GetSourceDeploymentId())
		require.Equal(f.t, targetID, req.GetTargetDeploymentId())
	case <-time.After(10 * time.Second):
		f.t.Fatal("expected rollback workflow invocation")
	}
}

func requireConnectError(t *testing.T, err error, code connect.Code, msgSubstr string) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, code, connect.CodeOf(err), "unexpected connect code, err: %v", err)
	require.ErrorContains(t, err, msgSubstr)
}

func TestDeployment_Promote_Validation(t *testing.T) {
	f := newLifecycleFixture(t)

	// arrange seeds the scenario and returns the deployment id to promote.
	rejections := []struct {
		name    string
		code    connect.Code
		message string
		arrange func() string
	}{
		{"deployment not found", connect.CodeNotFound, "deployment not found", func() string {
			return uid.New("deployment")
		}},
		{"not ready", connect.CodeFailedPrecondition, "deployment is not ready", func() string {
			return f.deployment(f.prodEnv, db.DeploymentsStatusPending, db.DeploymentsDesiredStateRunning).ID
		}},
		{"shutting down", connect.CodeFailedPrecondition, "deployment is shutting down", func() string {
			return f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateStopped).ID
		}},
		{"non-production", connect.CodeFailedPrecondition, "only production deployments can be promoted", func() string {
			return f.deployment(f.previewEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning).ID
		}},
		{"app has no live deployment", connect.CodeFailedPrecondition, "app has no live deployment", func() string {
			f.setLive("", false)
			return f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning).ID
		}},
		{"already live", connect.CodeFailedPrecondition, "deployment is already live", func() string {
			d := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
			f.setLive(d.ID, false)
			return d.ID
		}},
	}
	for _, tc := range rejections {
		t.Run(tc.name, func(t *testing.T) {
			requireConnectError(t, f.promote(tc.arrange()), tc.code, tc.message)
		})
	}

	// Promoting the current live deployment is allowed only when it confirms a
	// rollback (is_rolled_back = true).
	t.Run("promotes a rolled-back live deployment", func(t *testing.T) {
		d := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
		f.setLive(d.ID, true)
		require.NoError(t, f.promote(d.ID))
		f.requirePromoteWorkflow(d.ID)
	})

	t.Run("promotes a ready deployment over the live one", func(t *testing.T) {
		live := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
		f.setLive(live.ID, false)
		target := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
		require.NoError(t, f.promote(target.ID))
		f.requirePromoteWorkflow(target.ID)
	})
}

func TestDeployment_Rollback_Validation(t *testing.T) {
	f := newLifecycleFixture(t)

	// arrange seeds the scenario and returns the source and target ids.
	rejections := []struct {
		name    string
		code    connect.Code
		message string
		arrange func() (source, target string)
	}{
		{"source equals target", connect.CodeFailedPrecondition, "must be different", func() (string, string) {
			d := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
			return d.ID, d.ID
		}},
		{"source not found", connect.CodeNotFound, "source deployment not found", func() (string, string) {
			return uid.New("deployment"), f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning).ID
		}},
		{"target not found", connect.CodeNotFound, "target deployment not found", func() (string, string) {
			return f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning).ID, uid.New("deployment")
		}},
		{"different environment", connect.CodeFailedPrecondition, "same environment", func() (string, string) {
			return f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning).ID,
				f.deployment(f.previewEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning).ID
		}},
		{"target not ready", connect.CodeFailedPrecondition, "target deployment is not ready", func() (string, string) {
			source := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
			target := f.deployment(f.prodEnv, db.DeploymentsStatusPending, db.DeploymentsDesiredStateRunning)
			f.setLive(source.ID, false)
			return source.ID, target.ID
		}},
		{"target shutting down", connect.CodeFailedPrecondition, "target deployment is shutting down", func() (string, string) {
			source := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
			target := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateStopped)
			f.setLive(source.ID, false)
			return source.ID, target.ID
		}},
		{"non-production", connect.CodeFailedPrecondition, "only production deployments can be rolled back", func() (string, string) {
			source := f.deployment(f.previewEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
			target := f.deployment(f.previewEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
			f.setLive(source.ID, false)
			return source.ID, target.ID
		}},
		{"source not current live", connect.CodeFailedPrecondition, "source deployment is not the current live deployment", func() (string, string) {
			source := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
			target := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
			f.setLive(target.ID, false)
			return source.ID, target.ID
		}},
	}
	for _, tc := range rejections {
		t.Run(tc.name, func(t *testing.T) {
			source, target := tc.arrange()
			requireConnectError(t, f.rollback(source, target), tc.code, tc.message)
		})
	}

	t.Run("rolls back the live deployment to a previous one", func(t *testing.T) {
		source := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
		target := f.deployment(f.prodEnv, db.DeploymentsStatusReady, db.DeploymentsDesiredStateRunning)
		f.setLive(source.ID, false)
		require.NoError(t, f.rollback(source.ID, target.ID))
		f.requireRollbackWorkflow(source.ID, target.ID)
	})
}
