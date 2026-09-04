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

// observedPromotion contains the Restate object key and typed request.
type observedPromotion struct {
	virtualObjectKey string
	request          *hydrav1.PromoteDeploymentRequest
}

// recordingEnvironmentService captures PromoteDeployment invocations for assertions.
type recordingEnvironmentService struct {
	hydrav1.UnimplementedEnvironmentServiceServer
	promotions chan observedPromotion
}

// PromoteDeployment records the object key and typed payload received through Restate.
func (service *recordingEnvironmentService) PromoteDeployment(ctx restate.ObjectContext, request *hydrav1.PromoteDeploymentRequest) (*hydrav1.PromoteDeploymentResponse, error) {
	service.promotions <- observedPromotion{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.PromoteDeploymentResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose EnvironmentService
// reports each PromoteDeployment invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedPromotion) {
	t.Helper()

	recorder := &recordingEnvironmentService{
		promotions: make(chan observedPromotion, 1),
	}
	restateConfig := containers.Restate(t, hydrav1.NewEnvironmentServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.promotions
}

// newUncalledRestate fails during cleanup if EnvironmentService was invoked.
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, calls := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, calls, time.Second) })
	return client
}
