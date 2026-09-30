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

// observedWakeDeployment contains the Restate object key and typed request.
type observedWakeDeployment struct {
	virtualObjectKey string
	request          *hydrav1.WakeDeploymentRequest
}

// recordingDeploymentService captures WakeDeployment invocations for assertions.
type recordingDeploymentService struct {
	hydrav1.UnimplementedDeploymentServiceServer
	wakes chan observedWakeDeployment
}

// WakeDeployment records the object key and typed payload received through Restate.
func (service *recordingDeploymentService) WakeDeployment(ctx restate.ObjectContext, request *hydrav1.WakeDeploymentRequest) (*hydrav1.WakeDeploymentResponse, error) {
	service.wakes <- observedWakeDeployment{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.WakeDeploymentResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose DeploymentService
// reports each WakeDeployment invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedWakeDeployment) {
	t.Helper()

	recorder := &recordingDeploymentService{
		wakes: make(chan observedWakeDeployment, 1),
	}
	restateConfig := containers.Restate(t, hydrav1.NewDeploymentServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.wakes
}

// newUncalledRestate fails during cleanup if DeploymentService was invoked.
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, calls := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, calls, time.Second) })
	return client
}
