package handler_test

import (
	"testing"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// observedProjectDelete contains the Restate object key separately from the
// typed request because the object key is encoded in the ingress path.
type observedProjectDelete struct {
	virtualObjectKey string
	request          *hydrav1.DeleteProjectRequest
}

// recordingProjectService captures Delete invocations for route assertions.
type recordingProjectService struct {
	hydrav1.UnimplementedProjectServiceServer
	deletes chan observedProjectDelete
}

// Delete records the object key and typed payload received through Restate.
func (service *recordingProjectService) Delete(ctx restate.ObjectContext, request *hydrav1.DeleteProjectRequest) (*hydrav1.DeleteProjectResponse, error) {
	service.deletes <- observedProjectDelete{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.DeleteProjectResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose ProjectService
// reports each Delete invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedProjectDelete) {
	t.Helper()

	recorder := &recordingProjectService{
		deletes: make(chan observedProjectDelete, 1),
	}
	restateConfig := containers.Restate(t, hydrav1.NewProjectServiceServer(recorder))

	return restateConfig.IngressClient, recorder.deletes
}
