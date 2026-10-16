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

type observedCancellation struct {
	virtualObjectKey string
	request          *hydrav1.CancelDeploymentRequest
}

type recordingDeploymentService struct {
	hydrav1.UnimplementedDeploymentServiceServer
	cancellations chan observedCancellation
}

func (service *recordingDeploymentService) CancelDeployment(ctx restate.ObjectContext, request *hydrav1.CancelDeploymentRequest) (*hydrav1.CancelDeploymentResponse, error) {
	service.cancellations <- observedCancellation{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.CancelDeploymentResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose DeploymentService
// reports each CancelDeployment invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedCancellation) {
	t.Helper()

	recorder := &recordingDeploymentService{
		cancellations: make(chan observedCancellation, 16),
	}
	restateConfig := containers.Restate(t, hydrav1.NewDeploymentServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.cancellations
}

// newUncalledRestate fails during cleanup if DeploymentService was invoked.
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, calls := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, calls, time.Second) })
	return client
}
