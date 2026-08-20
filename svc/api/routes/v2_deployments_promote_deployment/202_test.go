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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_promote_deployment"
)

func TestPromoteDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, promotions := newRecordingRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.promote_deployment"},
	})

	live := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	setCurrentDeployment(t, h, setup.App.ID, live.ID)

	target := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: target.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	rootKeyID, err := db.Query.FindKeyIDByHash(context.Background(), h.DB.RO(), hash.Sha256(setup.RootKey))
	require.NoError(t, err)

	observed := testutil.Receive(t, promotions, 10*time.Second)
	require.Equal(t, target.ID, observed.virtualObjectKey)
	require.Equal(t, target.ID, observed.request.GetTargetDeploymentId())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
	require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
	require.NotEmpty(t, observed.request.GetCorrelationId())
}

// Promoting the live deployment while the app is rolled back confirms the
// rollback, so it must be allowed through to ctrl.
func TestPromoteDeploymentConfirmRollback(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, promotions := newRecordingRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.promote_deployment"},
	})

	live := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	markRolledBack(t, h, setup.App.ID, live.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: live.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	observed := testutil.Receive(t, promotions, 10*time.Second)
	require.Equal(t, live.ID, observed.virtualObjectKey)
	require.Equal(t, live.ID, observed.request.GetTargetDeploymentId())
}
