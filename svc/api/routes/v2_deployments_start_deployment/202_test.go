package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_start_deployment"
)

func TestStartDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, wakes := newRecordingRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.start_deployment"},
	})

	preview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
		Description: "preview environment",
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusStopped,
		DesiredState:  mysqltype.DeploymentsDesiredStateStopped,
		GitBranch:     "KEBAP",
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	rootKeyID, err := db.Query.FindKeyIDByHash(context.Background(), h.DB.RO(), hash.Sha256(setup.RootKey))
	require.NoError(t, err)

	observed := testutil.Receive(t, wakes, 10*time.Second)
	require.Equal(t, dep.ID, observed.virtualObjectKey)
	require.Equal(t, dep.ID, observed.request.GetDeploymentId())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
	require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
	require.NotEmpty(t, observed.request.GetCorrelationId())
}

// Stopping sets desired_state=stopped immediately while status stays ready
// until krane drains the last instance. Start keys on the intent, so a
// deployment still draining is wakeable.
func TestStartDeploymentWhileDraining(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, wakes := newRecordingRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.start_deployment"},
	})

	preview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
		Description: "preview environment",
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		DesiredState:  mysqltype.DeploymentsDesiredStateStopped,
		GitBranch:     "KEBAP",
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	observed := testutil.Receive(t, wakes, 10*time.Second)
	require.Equal(t, dep.ID, observed.virtualObjectKey)
	require.Equal(t, dep.ID, observed.request.GetDeploymentId())
}

// The environment-scoped permission must work as well as the wildcard.
func TestStartDeploymentScopedPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, wakes := newRecordingRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()

	preview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
		Description: "preview environment",
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusStopped,
		DesiredState:  mysqltype.DeploymentsDesiredStateStopped,
	})

	rootKey := h.CreateRootKey(setup.Workspace.ID, "environment."+preview.ID+".start_deployment")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
	observed := testutil.Receive(t, wakes, 10*time.Second)
	require.Equal(t, dep.ID, observed.virtualObjectKey)
	require.Equal(t, dep.ID, observed.request.GetDeploymentId())
}
