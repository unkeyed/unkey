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
	request          *hydrav1.PromoteRequest
}

// recordingDeployService captures Promote invocations for assertions.
type recordingDeployService struct {
	hydrav1.UnimplementedDeployServiceServer
	promotions chan observedPromotion
}

// Promote records the object key and typed payload received through Restate.
func (service *recordingDeployService) Promote(ctx restate.ObjectContext, request *hydrav1.PromoteRequest) (*hydrav1.PromoteResponse, error) {
	service.promotions <- observedPromotion{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.PromoteResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose DeployService
// reports each Promote invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedPromotion) {
	t.Helper()

	recorder := &recordingDeployService{
		promotions: make(chan observedPromotion, 1),
	}
	restateConfig := containers.Restate(t, hydrav1.NewDeployServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.promotions
}

// newUncalledRestate fails during cleanup if DeployService was invoked.
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, calls := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, calls, time.Second) })
	return client
}
