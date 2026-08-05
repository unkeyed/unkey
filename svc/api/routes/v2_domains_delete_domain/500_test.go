package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_delete_domain"
)

// TestDeleteDomainCtrlFailureIsRetryable pins the client-visible half of ctrl's
// transactional delete: when ctrl fails internally its transaction rolled back and
// the domain still exists, so the API surfaces a 500 and retrying the identical
// request succeeds once ctrl recovers.
func TestDeleteDomainCtrlFailureIsRetryable(t *testing.T) {
	h := testutil.NewHarness(t)

	failuresLeft := 1
	ctrlClient := &testutil.MockCustomDomainClient{
		DeleteCustomDomainFunc: func(_ context.Context, _ *ctrlv1.DeleteCustomDomainRequest) (*ctrlv1.DeleteCustomDomainResponse, error) {
			if failuresLeft > 0 {
				failuresLeft--
				return nil, connect.NewError(connect.CodeInternal, errors.New("failed to delete custom domain"))
			}
			return &ctrlv1.DeleteCustomDomainResponse{}, nil
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.delete_domain")
	req := handler.Request{Domain: seeded.domain}

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "Failed to delete custom domain.", res.Body.Error.Detail,
		"ctrl's internal wording must not leak, received: %s", res.RawBody)

	retry := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusOK, retry.Status,
		"the failed delete left the domain in place, so the same request must succeed, received: %s", retry.RawBody)
	require.Len(t, ctrlClient.DeleteCustomDomainCalls, 2)
}

// TestDeleteDomainCtrlUnreachable covers the transport failure: no connect code at
// all, just a dead control plane. Same contract, a 500 with a stable message.
func TestDeleteDomainCtrlUnreachable(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{
		DeleteCustomDomainFunc: func(_ context.Context, _ *ctrlv1.DeleteCustomDomainRequest) (*ctrlv1.DeleteCustomDomainResponse, error) {
			return nil, errors.New("dial tcp: connection refused")
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.delete_domain")

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "Failed to delete custom domain.", res.Body.Error.Detail,
		"transport detail must not leak, received: %s", res.RawBody)
}
