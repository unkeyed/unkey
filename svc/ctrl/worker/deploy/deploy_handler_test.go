package deploy

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestSpinDownStopsConnectedPreviewAfterOneMinute(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "project", Slug: slug(uid.ProjectPrefix),
	})
	newPreview := func() (string, string) {
		t.Helper()
		app := seeder.CreateApp(ctx, seed.CreateAppRequest{
			ID: uid.New(uid.AppPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, Name: "app", Slug: slug(uid.AppPrefix),
		})
		environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
			ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID,
			Slug: "preview", Kind: mysqltype.EnvironmentKindPreview,
		})
		return app.ID, environment.ID
	}
	deploy := func(appID, environmentID string) string {
		t.Helper()
		return seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			WorkspaceID: workspaceID, ProjectID: project.ID, AppID: appID, EnvironmentID: environmentID,
			Status: mysqltype.DeploymentsStatusReady, Source: db.DeploymentsSourceGit,
			GitBranch:    sql.NullString{String: "feature", Valid: true},
			Capabilities: mysqltype.DeploymentCapabilities{PrivateNetworking: true},
		}).ID
	}
	targetAppID, targetEnvironmentID := newPreview()
	callerAppID, callerEnvironmentID := newPreview()
	replaced := deploy(targetAppID, targetEnvironmentID)
	caller := deploy(callerAppID, callerEnvironmentID)

	connectionID := seeder.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
		WorkspaceID: workspaceID, ProjectID: project.ID, CallerAppID: callerAppID, CallerEnvironmentID: callerEnvironmentID,
		TargetAppID: targetAppID, Name: "target",
	})
	require.NoError(t, database.InsertDeploymentConnection(ctx, db.InsertDeploymentConnectionParams{
		DeploymentID: caller, ConnectionID: connectionID, WorkspaceID: workspaceID, ProjectID: project.ID,
		AppID: callerAppID, EnvironmentID: callerEnvironmentID, ResourceType: "app", ResourceID: targetAppID,
		Name: "target", CreatedAt: time.Now().UnixMilli(),
	}))
	require.NoError(t, database.InsertDeploymentConnectionAppTarget(ctx, db.InsertDeploymentConnectionAppTargetParams{
		DeploymentID: caller, ConnectionID: connectionID,
		SelectionMode: db.DeploymentConnectionAppTargetsSelectionModeAutomatic,
	}))
	replacement := deploy(targetAppID, targetEnvironmentID)

	stops := &scheduledStops{requests: map[string]*hydrav1.ScheduleDesiredStateChangeRequest{}}
	cfg := containers.Restate(t,
		restate.Reflect(SpinDownProbe{workflow: &Workflow{db: database}}),
		hydrav1.NewDeploymentServiceServer(stops),
	)
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err = ingress.Object[string, string](cfg.IngressClient, "SpinDownProbe", replacement, "SpinDown").Request(callCtx, replacement)
	require.NoError(t, err)

	stops.mu.Lock()
	defer stops.mu.Unlock()
	require.Len(t, stops.requests, 1)
	stop := stops.requests[replaced]
	require.NotNil(t, stop, "the replaced preview must be scheduled to stop")
	require.Equal(t, int64(60_000), stop.GetDelayMillis(), "an app connection must not extend the one-minute preview replacement delay")
	require.Equal(t, hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED, stop.GetState())
	require.False(t, stop.GetOverwrite())
	require.True(t, stop.GetDeferWhilePinned())
}

type SpinDownProbe struct {
	workflow *Workflow
}

func (p SpinDownProbe) SpinDown(ctx restate.ObjectContext, deploymentID string) (string, error) {
	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindDeploymentForDeployRow, error) {
		return p.workflow.db.FindDeploymentForDeploy(runCtx, deploymentID)
	})
	if err != nil {
		return "", err
	}
	return "", p.workflow.spinDownPreviousDeployments(ctx, deployment)
}

type scheduledStops struct {
	hydrav1.UnimplementedDeploymentServiceServer
	mu       sync.Mutex
	requests map[string]*hydrav1.ScheduleDesiredStateChangeRequest
}

func (s *scheduledStops) ScheduleDesiredStateChange(ctx restate.ObjectContext, req *hydrav1.ScheduleDesiredStateChangeRequest) (*hydrav1.ScheduleDesiredStateChangeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests[restate.Key(ctx)] = req
	return &hydrav1.ScheduleDesiredStateChangeResponse{}, nil
}

func TestSkippedPromotionPreservesConnectionPins(t *testing.T) {
	mockCtx := mocks.NewMockContext(t)
	routing := mocks.NewMockClient(t)
	deployment := mocks.NewMockClient(t)
	routingResult := mocks.NewMockResponseFuture(t)
	deploymentResult := mocks.NewMockResponseFuture(t)
	environmentID, olderID := uid.New(uid.EnvironmentPrefix), uid.New(uid.DeploymentPrefix)
	mockCtx.On("Object", "hydra.v1.RoutingService", environmentID, "SwapLiveDeployment", mock.Anything).Return(routing)
	mockCtx.On("Object", "hydra.v1.DeploymentService", olderID, "ScheduleDesiredStateChange", mock.Anything).Return(deployment)
	routing.On("RequestFuture", mock.Anything).Return(routingResult).Once()
	routingResult.On("Response", mock.Anything).Run(func(args mock.Arguments) {
		response := args.Get(0).(**hydrav1.SwapLiveDeploymentResponse)
		*response = &hydrav1.SwapLiveDeploymentResponse{
			AutomaticPromotionSkipReason: hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_NEWER_DEPLOYMENT,
		}
	}).Return(nil).Once()
	deployment.On("RequestFuture", mock.MatchedBy(func(req *hydrav1.ScheduleDesiredStateChangeRequest) bool {
		return req.GetDeferWhilePinned() && req.GetOverwrite() && req.GetDelayMillis() == 0 &&
			req.GetState() == hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED
	}), mock.Anything).Return(deploymentResult).Once()
	deploymentResult.On("Response", mock.Anything).Return(nil).Once()

	var workflow Workflow
	reason, err := workflow.swapLiveDeployment(restate.WithMockContext(mockCtx), db.FindDeploymentForDeployRow{
		ID:            olderID,
		EnvironmentID: environmentID,
	}, nil)
	require.NoError(t, err)
	require.Equal(t, hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_NEWER_DEPLOYMENT, reason)
}
