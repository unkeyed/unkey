package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_create_domain"
)

// TestCreateDomainCtrlFailureIsRetryable pins the client-visible half of ctrl's
// all-or-nothing create: when ctrl fails internally (its transaction rolled back,
// nothing was written), the API surfaces a 500, and retrying the identical
// request succeeds once ctrl recovers — rather than a 409 pointing at a domain
// that was never attached.
func TestCreateDomainCtrlFailureIsRetryable(t *testing.T) {
	h := testutil.NewHarness(t)

	failuresLeft := 1
	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, _ *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			if failuresLeft > 0 {
				failuresLeft--
				return nil, connect.NewError(connect.CodeInternal, errors.New("failed to trigger verification workflow"))
			}
			return &ctrlv1.AddCustomDomainResponse{
				DomainId:          uid.New(uid.DomainPrefix),
				TargetCname:       "a1b2c3d4e5f6g7h8.cname.unkey.com",
				VerificationToken: "3ZQ8xK1mP7vT5nR2wY6bJ4hL",
			}, nil
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")
	req := makeRequest(env, randomDomain())

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "Failed to create custom domain.", res.Body.Error.Detail,
		"ctrl's internal wording must not leak, received: %s", res.RawBody)

	retry := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusOK, retry.Status,
		"the failed create left nothing behind, so the same request must succeed, received: %s", retry.RawBody)
	require.Len(t, ctrlClient.AddCustomDomainCalls, 2)
}

// TestCreateDomainCtrlUnreachable covers the transport failure: no connect code at
// all, just a dead control plane. Same contract, a 500 with a stable message.
func TestCreateDomainCtrlUnreachable(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, _ *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return nil, errors.New("dial tcp: connection refused")
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(rootKey), makeRequest(env, randomDomain()))
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "Failed to create custom domain.", res.Body.Error.Detail,
		"transport detail must not leak, received: %s", res.RawBody)
}
