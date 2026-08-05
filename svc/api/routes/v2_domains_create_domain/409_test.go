package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_create_domain"
)

// TestCreateDomainDuplicate covers the race: the handler's availability check
// passed and ctrl rejected the name anyway. The mock sends what ctrl sends, which is
// gatefault carrying the gate's public message. Inventing a message here would let
// ctrl's internal wording reach callers with no test noticing.
func TestCreateDomainDuplicate(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, req *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return nil, connect.NewError(
				connect.CodeAlreadyExists,
				errors.New(fault.UserFacingMessage(domaingate.AlreadyExists(req.GetDomain()))),
			)
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	domain := randomDomain()
	res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, authHeaders(rootKey), makeRequest(env, domain))
	require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/domain_already_exists", res.Body.Error.Type)

	// Pinned in full, not just for the domain: a message naming the domain is not
	// necessarily a message the caller can act on.
	require.Equal(t,
		fmt.Sprintf("The domain '%s' is already attached to this workspace.", domain),
		res.Body.Error.Detail,
		"received: %s", res.RawBody)
}
