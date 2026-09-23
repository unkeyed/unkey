package deploy

import (
	"context"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestNotifyReadinessFailsLegacyWaiter(t *testing.T) {
	w := &Workflow{}
	service := restate.NewWorkflow("LegacyReadiness").
		Handler("Run", restate.NewWorkflowHandler(func(ctx restate.WorkflowContext, _ restate.Void) (restate.Void, error) {
			return restate.Promise[restate.Void](ctx, instancesReadyPromise).Result()
		})).
		Handler("Notify", restate.NewWorkflowSharedHandler(w.NotifyReadiness))
	runtime := containers.Restate(t, service)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(cancel)
	key := uid.New("readiness")
	_, err := ingress.Workflow[*hydrav1.NotifyReadinessRequest, *hydrav1.NotifyReadinessResponse](runtime.IngressClient, "LegacyReadiness", key, "Notify").Request(ctx, &hydrav1.NotifyReadinessRequest{State: hydrav1.NotifyReadinessRequest_STATE_OUT_OF_MEMORY})
	require.NoError(t, err)
	_, err = ingress.Workflow[restate.Void, restate.Void](runtime.IngressClient, "LegacyReadiness", key, "Run").Request(ctx, restate.Void{})
	require.ErrorContains(t, err, "out_of_memory")
}

func TestNotifyReadinessKeepsReadyBeforeFailure(t *testing.T) {
	w := &Workflow{}
	service := restate.NewWorkflow("ReadinessOrdering").
		Handler("Run", restate.NewWorkflowHandler(func(ctx restate.WorkflowContext, _ restate.Void) (restate.Void, error) {
			return restate.Promise[restate.Void](ctx, instancesReadyPromise).Result()
		})).
		Handler("Notify", restate.NewWorkflowSharedHandler(w.NotifyReadiness))
	runtime := containers.Restate(t, service)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(cancel)
	key := uid.New("readiness")
	for _, state := range []hydrav1.NotifyReadinessRequest_State{
		hydrav1.NotifyReadinessRequest_STATE_READY,
		hydrav1.NotifyReadinessRequest_STATE_OUT_OF_MEMORY,
	} {
		_, err := ingress.Workflow[*hydrav1.NotifyReadinessRequest, *hydrav1.NotifyReadinessResponse](runtime.IngressClient, "ReadinessOrdering", key, "Notify").Request(ctx, &hydrav1.NotifyReadinessRequest{State: state})
		require.NoError(t, err)
	}
	_, err := ingress.Workflow[restate.Void, restate.Void](runtime.IngressClient, "ReadinessOrdering", key, "Run").Request(ctx, restate.Void{})
	require.NoError(t, err, "the first readiness completion must survive a later failure")
}
