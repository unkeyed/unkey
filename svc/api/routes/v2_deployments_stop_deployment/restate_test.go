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

// observedStopDeployment contains the Restate object key and typed request.
type observedStopDeployment struct {
	virtualObjectKey string
	request          *hydrav1.StopDeploymentRequest
}

// recordingDeploymentService captures StopDeployment invocations for assertions.
type recordingDeploymentService struct {
	hydrav1.UnimplementedDeploymentServiceServer
	stops chan observedStopDeployment
}

// StopDeployment records the object key and typed payload received through Restate.
func (service *recordingDeploymentService) StopDeployment(ctx restate.ObjectContext, request *hydrav1.StopDeploymentRequest) (*hydrav1.StopDeploymentResponse, error) {
	service.stops <- observedStopDeployment{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.StopDeploymentResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose DeploymentService
// reports each StopDeployment invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedStopDeployment) {
	t.Helper()

	recorder := &recordingDeploymentService{
		stops: make(chan observedStopDeployment, 1),
	}
	restateConfig := containers.Restate(t, hydrav1.NewDeploymentServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.stops
}

// newUncalledRestate fails during cleanup if DeploymentService was invoked.
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, calls := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, calls, time.Second) })
	return client
}
