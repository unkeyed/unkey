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
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_verify_domain"
)

// TestVerifyDomainAlreadyVerified pins the local gate: a verified domain is rejected
// before ctrl is involved, so a serving domain can never have its workflow restarted
// by this endpoint.
func TestVerifyDomainAlreadyVerified(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, verifiedDomain)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, fmt.Sprintf("The domain '%s' is already verified. No action is needed.", seeded.domain), res.Body.Error.Detail)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/precondition_failed", res.Body.Error.Type)
	require.Empty(t, ctrlClient.RetryVerificationCalls,
		"a verified domain must never reach ctrl")
}

// TestVerifyDomainCtrlPreconditionIsMasked covers the race the advisory read leaves
// open: the row read as retryable on the replica but ctrl found it verified on the
// primary. The caller gets the same 412 as the local gate, with a fixed message
// rather than ctrl's internal wording.
func TestVerifyDomainCtrlPreconditionIsMasked(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{
		RetryVerificationFunc: func(_ context.Context, req *ctrlv1.RetryVerificationRequest) (*ctrlv1.RetryVerificationResponse, error) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("internal gate wording: "+req.GetDomain()))
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "The domain is already verified. No action is needed.", res.Body.Error.Detail,
		"ctrl's internal wording must not leak, received: %s", res.RawBody)
	require.Len(t, ctrlClient.RetryVerificationCalls, 1)
}
