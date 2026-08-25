package handler_test

import (
	"testing"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// observedAppDelete contains the Restate object key separately from the typed
// request because the object key is encoded in the ingress path.
type observedAppDelete struct {
	virtualObjectKey string
	request          *hydrav1.DeleteAppRequest
}

// recordingAppService captures Delete invocations for route assertions.
type recordingAppService struct {
	hydrav1.UnimplementedAppServiceServer
	deletes chan observedAppDelete
}

// Delete records the object key and typed payload received through Restate.
func (service *recordingAppService) Delete(ctx restate.ObjectContext, request *hydrav1.DeleteAppRequest) (*hydrav1.DeleteAppResponse, error) {
	service.deletes <- observedAppDelete{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.DeleteAppResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose AppService
// reports each Delete invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedAppDelete) {
	t.Helper()

	recorder := &recordingAppService{
		deletes: make(chan observedAppDelete, 1),
	}
	restateConfig := containers.Restate(t, hydrav1.NewAppServiceServer(recorder))

	return restateConfig.IngressClient, recorder.deletes
}
