package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_authorize_deployment"
)

func TestAuthorizeDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	restateClient, authorizations := newRecordingRestate(t)
	route := newRoute(h, restateClient)

	setup := h.CreateTestDeploymentSetup()
	dep := seedDeployment(h, setup, mysqltype.DeploymentsStatusAwaitingApproval)
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentURN(setup, dep.ID, "write"))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	rootKeyID, err := db.Query.FindKeyIDByHash(t.Context(), h.DB.RO(), hash.Sha256(rootKey))
	require.NoError(t, err)

	observed := testutil.Receive(t, authorizations, 10*time.Second)
	require.Equal(t, dep.ID, observed.virtualObjectKey)
	require.Equal(t, dep.ID, observed.request.GetDeploymentId())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
	require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
	require.NotEmpty(t, observed.request.GetCorrelationId())
}
