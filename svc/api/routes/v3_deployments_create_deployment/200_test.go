package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_deployments_create_deployment"
)

// observedCreate contains the Restate object key, which is the deployment id the
// handler minted, and the typed request it submitted.
type observedCreate struct {
	virtualObjectKey string
	request          *hydrav1.DeployCreateRequest
}

type recordingDeployService struct {
	hydrav1.UnimplementedDeployServiceServer
	creates chan observedCreate
}

func (service *recordingDeployService) Create(ctx restate.ObjectContext, request *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	service.creates <- observedCreate{
		virtualObjectKey: restate.Key(ctx),
		request:          request,
	}
	return &hydrav1.DeployCreateResponse{Outcome: hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED}, nil
}

func newRecordingRestate(t *testing.T) (*restateingress.Client, <-chan observedCreate) {
	t.Helper()

	recorder := &recordingDeployService{
		UnimplementedDeployServiceServer: hydrav1.UnimplementedDeployServiceServer{},
		creates:                          make(chan observedCreate, 8),
	}
	restateConfig := containers.Restate(t, hydrav1.NewDeployServiceServer(recorder))

	return restateingress.NewClient(restateConfig.IngressURL), recorder.creates
}

// newUncalledRestate fails during cleanup if Create was invoked
func newUncalledRestate(t *testing.T) *restateingress.Client {
	t.Helper()

	client, creates := newRecordingRestate(t)
	t.Cleanup(func() { testutil.RequireNoReceive(t, creates, time.Second) })
	return client
}

func TestCreateOCIDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	restateClient, creates := newRecordingRestate(t)
	route := &handler.Handler{DB: h.DB, Restate: restateClient}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Data.DeploymentId)

	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, res.Body.Data.DeploymentId, observed.virtualObjectKey)
	require.Equal(t, "nginx:latest", observed.request.GetImage().GetImage())
	require.Nil(t, observed.request.GetGit())
}

func TestCreateDeploymentWithAppDefault(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	restateClient, creates := newRecordingRestate(t)
	route := &handler.Handler{DB: h.DB, Restate: restateClient}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)

	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Nil(t, observed.request.GetSource(), "no source override must leave the oneof unset")
}

func TestCreateGitDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	restateClient, creates := newRecordingRestate(t)
	route := &handler.Handler{DB: h.DB, Restate: restateClient}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Git:         &openapi.DeploymentSourceGit{Branch: ptr.P("main")},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)

	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, "main", observed.request.GetGit().GetCommit().GetBranch())
	require.Nil(t, observed.request.GetImage())
}

func TestRedeploySendsExistingDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	_, err := h.DB.RW().ExecContext(context.Background(), `
		UPDATE deployments
		SET source = ?, image_requested = ?, image_resolved = ?
		WHERE id = ?
	`, db.DeploymentsSourceOci, requested, resolved, deployment.ID)
	require.NoError(t, err)

	restateClient, creates := newRecordingRestate(t)
	route := &handler.Handler{DB: h.DB, Restate: restateClient}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Deployment:  &openapi.DeploymentSourceDeployment{DeploymentId: deployment.ID},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)

	// The worker resolves the image from the source row; the API only names it
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, deployment.ID, observed.request.GetExistingDeployment().GetDeploymentId())
	require.Nil(t, observed.request.GetImage())
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}
