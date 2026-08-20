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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_stop_deployment"
)

func TestStopDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	restateClient, stops := newRecordingRestate(t)
	route := newRoute(h, restateClient)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.stop_deployment"},
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
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	rootKeyID, err := db.Query.FindKeyIDByHash(context.Background(), h.DB.RO(), hash.Sha256(setup.RootKey))
	require.NoError(t, err)

	observed := testutil.Receive(t, stops, 10*time.Second)
	require.Equal(t, dep.ID, observed.virtualObjectKey)
	require.Equal(t, dep.ID, observed.request.GetDeploymentId())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
	require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
	require.NotEmpty(t, observed.request.GetCorrelationId())
}
