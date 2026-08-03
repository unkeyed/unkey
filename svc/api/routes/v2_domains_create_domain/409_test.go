package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_create_domain"
)

func TestCreateDomainDuplicate(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, req *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("domain already registered: %s", req.GetDomain()))
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	domain := randomDomain()
	res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, authHeaders(rootKey), makeRequest(env, domain))
	require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/domain_already_exists", res.Body.Error.Type)

	// The message must name the colliding domain so the caller knows which input
	// to change without a second request.
	require.Contains(t, res.Body.Error.Detail, domain, "expected the detail to name the domain, received: %s", res.RawBody)
}
