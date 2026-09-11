package handler_test

import (
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// observedRollback contains the Restate object key and typed request.
type observedRollback struct {
	virtualObjectKey string
	request          *hydrav1.RollbackDeploymentRequest
}

// recordingEnvironmentService captures RollbackDeployment invocations for assertions.
type recordingEnvironmentService struct {
	hydrav1.UnimplementedEnvironmentServiceServer
	rollbacks chan observedRollback
}

// RollbackDeployment records the object key and typed payload received through Restate.
func (service *recordingEnvironmentService) RollbackDeployment(ctx restate.ObjectContext, request *hydrav1.RollbackDeploymentRequest) (*hydrav1.RollbackDeploymentResponse, error) {
	service.rollbacks <- observedRollback{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.RollbackDeploymentResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose EnvironmentService
// reports each RollbackDeployment invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedRollback) {
	t.Helper()

	recorder := &recordingEnvironmentService{
		rollbacks: make(chan observedRollback, 1),
	}
	restateConfig := containers.Restate(t, hydrav1.NewEnvironmentServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.rollbacks
}

// newUncalledRestate fails during cleanup if EnvironmentService was invoked.
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, calls := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, calls, time.Second) })
	return client
}
