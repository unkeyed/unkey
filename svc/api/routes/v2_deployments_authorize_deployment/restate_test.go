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

type observedAuthorization struct {
	virtualObjectKey string
	request          *hydrav1.AuthorizeDeploymentRequest
}

type recordingDeploymentService struct {
	hydrav1.UnimplementedDeploymentServiceServer
	authorizations chan observedAuthorization
}

func (service *recordingDeploymentService) AuthorizeDeployment(ctx restate.ObjectContext, request *hydrav1.AuthorizeDeploymentRequest) (*hydrav1.AuthorizeDeploymentResponse, error) {
	service.authorizations <- observedAuthorization{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.AuthorizeDeploymentResponse{}, nil
}

// newRecordingRestate starts an isolated Restate server whose DeploymentService
// reports each AuthorizeDeployment invocation to the returned channel.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedAuthorization) {
	t.Helper()

	recorder := &recordingDeploymentService{
		authorizations: make(chan observedAuthorization, 16),
	}
	restateConfig := containers.Restate(t, hydrav1.NewDeploymentServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.authorizations
}

// newUncalledRestate fails during cleanup if DeploymentService was invoked.
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, calls := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, calls, time.Second) })
	return client
}
