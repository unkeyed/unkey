package deploy

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

func (w *Workflow) NotifyReadiness(ctx restate.WorkflowSharedContext, req *hydrav1.NotifyReadinessRequest) (*hydrav1.NotifyReadinessResponse, error) {
	var result readinessResult
	switch req.GetState() {
	case hydrav1.NotifyReadinessRequest_STATE_READY:
		result = readinessReady
	case hydrav1.NotifyReadinessRequest_STATE_OUT_OF_MEMORY:
		result = readinessOutOfMemory
	case hydrav1.NotifyReadinessRequest_STATE_CRASH_LOOP_BACK_OFF:
		result = readinessCrashLoopBackOff
	case hydrav1.NotifyReadinessRequest_STATE_CONTAINER_CANNOT_RUN:
		result = readinessContainerCannotRun
	case hydrav1.NotifyReadinessRequest_STATE_CONTAINER_CONFIG_ERROR:
		return &hydrav1.NotifyReadinessResponse{}, nil
	case hydrav1.NotifyReadinessRequest_STATE_INVALID_IMAGE_NAME:
		result = readinessInvalidImageName
	case hydrav1.NotifyReadinessRequest_STATE_UNSPECIFIED:
		fallthrough
	default:
		return nil, restate.ToTerminalError(fmt.Errorf("unsupported readiness state %s", req.GetState()), restate.WithErrorCode(400))
	}
	promise := restate.Promise[restate.Void](ctx, instancesReadyPromise)
	var err restate.TerminalError
	if result == readinessReady {
		err = promise.Resolve(restate.Void{})
	} else {
		err = promise.Reject(restate.TerminalErrorf("%s", result))
	}
	if err != nil && !(err.Code() == 409 && err.Message() == "promise was already completed") {
		return nil, err
	}
	return &hydrav1.NotifyReadinessResponse{}, nil
}
