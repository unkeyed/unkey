package testutil

import (
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// ObservedCreate is one DeployService.Create call a handler made: the
// deployment id it ran under and the request it carried.
type ObservedCreate struct {
	DeploymentID string
	Request      *hydrav1.DeployCreateRequest
}

type recordingDeployService struct {
	hydrav1.UnimplementedDeployServiceServer
	creates chan ObservedCreate
}

func (service *recordingDeployService) Create(ctx restate.ObjectContext, request *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	service.creates <- ObservedCreate{
		DeploymentID: restate.Key(ctx),
		Request:      request,
	}
	return &hydrav1.DeployCreateResponse{
		Outcome: hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
	}, nil
}

// RecordingDeployRestate starts an isolated Restate whose DeployService answers
// every Create as created and reports it on the returned channel.
func RecordingDeployRestate(t *testing.T) (*restateingress.Client, <-chan ObservedCreate) {
	t.Helper()

	recorder := &recordingDeployService{
		UnimplementedDeployServiceServer: hydrav1.UnimplementedDeployServiceServer{},
		creates:                          make(chan ObservedCreate, 8),
	}
	restateConfig := containers.Restate(t, hydrav1.NewDeployServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.creates
}

type rejectingDeployService struct {
	hydrav1.UnimplementedDeployServiceServer
	outcome hydrav1.CreateOutcome
}

func (service *rejectingDeployService) Create(_ restate.ObjectContext, _ *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	return &hydrav1.DeployCreateResponse{Outcome: service.outcome}, nil
}

// RejectingDeployRestate starts a Restate whose Create answers every call with
// outcome. The gates live in the worker and are tested there; a route test uses
// this to pin what a caller is told when one refuses.
func RejectingDeployRestate(t *testing.T, outcome hydrav1.CreateOutcome) *restateingress.Client {
	t.Helper()

	restateConfig := containers.Restate(t, hydrav1.NewDeployServiceServer(&rejectingDeployService{
		UnimplementedDeployServiceServer: hydrav1.UnimplementedDeployServiceServer{},
		outcome:                          outcome,
	}))

	return restateingress.NewClient(restateConfig.IngressURL)
}

// UncalledDeployRestate fails the test during cleanup if Create was invoked, for
// tests that must refuse before submitting.
func UncalledDeployRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, creates := RecordingDeployRestate(t)
	t.Cleanup(func() { RequireNoReceive(t, creates, time.Second) })
	return client
}
