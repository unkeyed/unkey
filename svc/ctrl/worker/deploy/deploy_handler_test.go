package deploy

import (
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestPreviewReplacementDelay(t *testing.T) {
	require.Equal(t, time.Minute, previewReplacementDelay(false))
	require.Equal(t, privatenetwork.ReplacementOverlap, previewReplacementDelay(true))
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
