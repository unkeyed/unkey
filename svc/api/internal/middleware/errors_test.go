package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// TestHTTPStatus_Mapping locks in the URN → HTTP status mapping for the
// api service. Most rows resolve via pkg/codes (Code.HTTPStatus); a few
// resolve via apiStatusOverrides (api treats missing/malformed auth as
// 400 Bad Request rather than 401, for backwards compatibility).
//
// A new URN that ends up returning 500 unintentionally — because either
// pkg/codes or this package forgot to map it — will fail this test.
func TestHTTPStatus_Mapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		urn        codes.URN
		wantStatus int
	}{
		// Per-service overrides (api maps these to 400, not 401):
		{codes.Auth.Authentication.Missing.URN(), http.StatusBadRequest},
		{codes.Auth.Authentication.Malformed.URN(), http.StatusBadRequest},
		{codes.Portal.Session.TokenMissing.URN(), http.StatusBadRequest},

		// Inherited from pkg/codes (User.BadRequest category → 400).
		{codes.User.BadRequest.PermissionsQuerySyntaxError.URN(), http.StatusBadRequest},
		{codes.User.BadRequest.RequestBodyUnreadable.URN(), http.StatusBadRequest},
		{codes.User.BadRequest.InvalidAnalyticsQuery.URN(), http.StatusBadRequest},

		// pkg/codes per-code overrides:
		{codes.User.BadRequest.RequestTimeout.URN(), http.StatusRequestTimeout},
		{codes.User.BadRequest.ClientClosedRequest.URN(), 499},
		{codes.User.BadRequest.RequestBodyTooLarge.URN(), http.StatusRequestEntityTooLarge},
		{codes.App.Validation.InvalidInput.URN(), http.StatusBadRequest},
		{codes.App.Precondition.PreconditionFailed.URN(), http.StatusPreconditionFailed},
		{codes.App.Protection.ProtectedResource.URN(), http.StatusPreconditionFailed},

		// CategoryUserUnprocessableEntity → 422
		{codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(), http.StatusUnprocessableEntity},
		{codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(), http.StatusUnprocessableEntity},
		{codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(), http.StatusUnprocessableEntity},

		// CategoryUserTooManyRequests → 429
		{codes.User.TooManyRequests.QueryQuotaExceeded.URN(), http.StatusTooManyRequests},
		{codes.User.TooManyRequests.WorkspaceRateLimited.URN(), http.StatusTooManyRequests},

		// CategoryUnkeyData → 404 by default.
		{codes.Data.Key.NotFound.URN(), http.StatusNotFound},
		{codes.Data.Workspace.NotFound.URN(), http.StatusNotFound},
		{codes.Data.Api.NotFound.URN(), http.StatusNotFound},
		{codes.Data.PortalConfig.NotFound.URN(), http.StatusNotFound},

		// CategoryUnkeyData per-code overrides:
		{codes.Data.Permission.Duplicate.URN(), http.StatusConflict},
		{codes.Data.Role.Duplicate.URN(), http.StatusConflict},
		{codes.Data.Identity.Duplicate.URN(), http.StatusConflict},
		{codes.Data.RatelimitNamespace.Gone.URN(), http.StatusGone},
		{codes.Data.Analytics.NotConfigured.URN(), http.StatusPreconditionFailed},
		{codes.Data.Analytics.ConnectionFailed.URN(), http.StatusServiceUnavailable},

		// CategoryUnkeyAuthentication → 401
		{codes.Auth.Authentication.KeyNotFound.URN(), http.StatusUnauthorized},
		{codes.Portal.Session.SessionNotFound.URN(), http.StatusUnauthorized},

		// CategoryUnkeyAuthorization → 403
		{codes.Auth.Authorization.Forbidden.URN(), http.StatusForbidden},
		{codes.Auth.Authorization.InsufficientPermissions.URN(), http.StatusForbidden},
		{codes.Auth.Authorization.KeyDisabled.URN(), http.StatusForbidden},
		{codes.Auth.Authorization.WorkspaceDisabled.URN(), http.StatusForbidden},

		// CategoryUnkeyApplication → 500 (genuine server-side faults).
		{codes.App.Internal.UnexpectedError.URN(), http.StatusInternalServerError},
		{codes.App.Internal.ServiceUnavailable.URN(), http.StatusInternalServerError},
		{codes.App.Validation.AssertionFailed.URN(), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.urn), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.wantStatus, httpStatus(tt.urn).Int(),
				"URN %s should map to HTTP %d", tt.urn, tt.wantStatus)
		})
	}
}

func TestHTTPStatus_UnknownURNDefaultsTo500(t *testing.T) {
	t.Parallel()

	require.Equal(t, http.StatusInternalServerError,
		httpStatus("err:made:up:nonexistent").Int(),
		"unknown URN should default to 500 so we get alerted on it")
}

// TestBuildErrorBody_PicksRightShape verifies that the body builder
// returns the correct openapi response type for each status. Drift
// here would mean clients receive a body shape that doesn't match the
// status code.
func TestBuildErrorBody_PicksRightShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status   codes.HTTPStatus
		wantType any
	}{
		{codes.StatusBadRequest, openapi.BadRequestErrorResponse{}},
		{codes.StatusRequestTimeout, openapi.BadRequestErrorResponse{}},
		{codes.StatusRequestEntityTooLarge, openapi.BadRequestErrorResponse{}},
		{codes.StatusClientClosedRequest, openapi.BadRequestErrorResponse{}},
		{codes.StatusUnauthorized, openapi.UnauthorizedErrorResponse{}},
		{codes.StatusForbidden, openapi.ForbiddenErrorResponse{}},
		{codes.StatusNotFound, openapi.NotFoundErrorResponse{}},
		{codes.StatusConflict, openapi.ConflictErrorResponse{}},
		{codes.StatusGone, openapi.GoneErrorResponse{}},
		{codes.StatusPreconditionFailed, openapi.PreconditionFailedErrorResponse{}},
		{codes.StatusUnprocessableEntity, openapi.UnprocessableEntityErrorResponse{}},
		{codes.StatusTooManyRequests, openapi.TooManyRequestsErrorResponse{}},
		{codes.StatusServiceUnavailable, openapi.ServiceUnavailableErrorResponse{}},
		{codes.StatusInternalServerError, openapi.InternalServerErrorResponse{}},
	}

	for _, tt := range cases {
		t.Run(tt.status.Text(), func(t *testing.T) {
			t.Parallel()
			body := buildErrorBody(tt.status, "Title", "https://docs", "detail", "req_xyz")
			require.IsType(t, tt.wantType, body)
		})
	}
}

func TestErrorTitle(t *testing.T) {
	t.Parallel()

	// 499 has no stdlib reason phrase — must come from override map.
	require.Equal(t, "Client Closed Request",
		errorTitle(codes.StatusClientClosedRequest, codes.User.BadRequest.ClientClosedRequest.URN()))

	// Standard reason phrase kicks in when no override exists.
	require.Equal(t, http.StatusText(http.StatusNotFound),
		errorTitle(codes.StatusNotFound, codes.Data.Key.NotFound.URN()))

	// API-specific phrasing for forbidden variants.
	require.Equal(t, "Insufficient Permissions",
		errorTitle(codes.StatusForbidden, codes.Auth.Authorization.InsufficientPermissions.URN()))
	require.Equal(t, "Key is disabled",
		errorTitle(codes.StatusForbidden, codes.Auth.Authorization.KeyDisabled.URN()))
}
