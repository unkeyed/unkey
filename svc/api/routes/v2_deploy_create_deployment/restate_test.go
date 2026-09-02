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

// observedCreate contains the Restate object key, which is the deployment id the
// handler minted, and the typed request it submitted.
type observedCreate struct {
	virtualObjectKey string
	request          *hydrav1.DeployCreateRequest
}

// recordingDeployService captures Create invocations for assertions.
type recordingDeployService struct {
	hydrav1.UnimplementedDeployServiceServer
	creates chan observedCreate
}

// Create records the object key and typed payload received through Restate.
func (service *recordingDeployService) Create(ctx restate.ObjectContext, request *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	service.creates <- observedCreate{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.DeployCreateResponse{
		Outcome:         hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
		RejectionReason: hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_UNSPECIFIED,
	}, nil
}

// newRecordingRestate starts an isolated Restate server whose DeployService
// reports each Create invocation to the returned channel. The handler submits
// the create one-way, so a test reads the channel to wait for it to land.
func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedCreate) {
	t.Helper()

	recorder := &recordingDeployService{
		creates: make(chan observedCreate, 8),
	}
	restateConfig := containers.Restate(t, hydrav1.NewDeployServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.creates
}

// newUncalledRestate fails during cleanup if Create was invoked. The validation,
// authorization, and gate tests use it to prove they refuse before submitting.
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, creates := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, creates, time.Second) })
	return client
}
