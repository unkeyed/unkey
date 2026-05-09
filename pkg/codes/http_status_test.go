package codes

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPStatus(t *testing.T) {
	t.Parallel()

	t.Run("valid status", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, HTTPStatus(http.StatusOK), NewHTTPStatus(200))
	})

	t.Run("low boundary", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, HTTPStatus(100), NewHTTPStatus(100))
	})

	t.Run("high boundary", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, HTTPStatus(599), NewHTTPStatus(599))
	})

	t.Run("panics below 100", func(t *testing.T) {
		t.Parallel()
		require.Panics(t, func() { NewHTTPStatus(99) })
	})

	t.Run("panics above 599", func(t *testing.T) {
		t.Parallel()
		require.Panics(t, func() { NewHTTPStatus(600) })
	})

	t.Run("panics on negative", func(t *testing.T) {
		t.Parallel()
		require.Panics(t, func() { NewHTTPStatus(-1) })
	})
}

func TestHTTPStatusText(t *testing.T) {
	t.Parallel()

	require.Equal(t, "Not Found", StatusNotFound.Text())
	require.Equal(t, "Unauthorized", StatusUnauthorized.Text())
	require.Equal(t, "Gateway Timeout", StatusGatewayTimeout.Text())
	// 499 is not registered with net/http, so Text() returns empty.
	require.Equal(t, "", StatusClientClosedRequest.Text())
}

func TestCategoryHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		category Category
		want     HTTPStatus
	}{
		{CategoryUserBadRequest, StatusBadRequest},
		{CategoryUserUnprocessableEntity, StatusUnprocessableEntity},
		{CategoryUserTooManyRequests, StatusTooManyRequests},
		{CategoryNotFound, StatusNotFound},
		{CategoryBadGateway, StatusBadGateway},
		{CategoryServiceUnavailable, StatusServiceUnavailable},
		{CategoryGatewayTimeout, StatusGatewayTimeout},
		{CategoryUnauthorized, StatusUnauthorized},
		{CategoryForbidden, StatusForbidden},
		{CategoryRateLimited, StatusTooManyRequests},
		{CategoryInternalServerError, StatusInternalServerError},
		{CategoryUnkeyAuthentication, StatusUnauthorized},
		{CategoryUnkeyAuthorization, StatusForbidden},
		{CategoryUnkeyLimits, StatusTooManyRequests},
		{CategoryUnkeyApplication, StatusInternalServerError},
		// Categories with no canonical default; codes within them
		// rely on Kind (Data) or are non-HTTP (Vault).
		{CategoryUnkeyData, 0},
		{CategoryUnkeyVault, 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.category.HTTPStatus())
		})
	}
}

// TestCategoryHTTPStatusCoverage walks every Category constant declared
// in this package and asserts that HTTPStatus() returns either a valid
// status or 0. New categories added without an HTTPStatus mapping will
// fail this test by triggering the default branch and returning 0
// outside the allowlist.
func TestCategoryHTTPStatusCoverage(t *testing.T) {
	t.Parallel()

	// Categories that legitimately have no canonical HTTP status.
	// Vault is non-HTTP (internal infrastructure); Data is too broad
	// for a single default and relies on per-code Kind instead.
	nonHTTPCategories := map[Category]bool{
		CategoryUnkeyVault: true,
		CategoryUnkeyData:  true,
	}

	for _, cat := range allCategories() {
		status := cat.HTTPStatus()
		if nonHTTPCategories[cat] {
			require.Equal(t, HTTPStatus(0), status,
				"non-HTTP category %s should return 0", cat)
			continue
		}
		require.NotEqual(t, HTTPStatus(0), status,
			"category %s has no HTTP status mapping; either add it to Category.HTTPStatus() or to nonHTTPCategories", cat)
		require.GreaterOrEqual(t, status.Int(), 100,
			"category %s returned invalid status %d", cat, status)
		require.LessOrEqual(t, status.Int(), 599,
			"category %s returned invalid status %d", cat, status)
	}
}

// TestCodeHTTPStatusCoverage walks every Code declared at package level
// (User, App, Auth, Data, Frontline, Portal) and asserts that every
// code resolves to a non-zero HTTPStatus, either via its Kind or via
// its Category default. Vault codes are exempt (non-HTTP subsystem).
//
// New data-style codes added without a Kind will fail here because
// CategoryUnkeyData has no canonical default.
func TestCodeHTTPStatusCoverage(t *testing.T) {
	t.Parallel()

	for _, code := range allCodes() {
		if code.Category == CategoryUnkeyVault {
			continue
		}
		require.NotEqual(t, HTTPStatus(0), code.HTTPStatus(),
			"code %s resolves to HTTPStatus 0; add a Kind or fix the Category",
			code.URN())
	}
}

// TestKindHTTPStatus locks in the wire-level mapping for every Kind
// the package defines. New Kind values without a status mapping fail
// here.
func TestKindHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind Kind
		want HTTPStatus
	}{
		{KindUnknown, 0},
		{KindNotFound, StatusNotFound},
		{KindDuplicate, StatusConflict},
		{KindGone, StatusGone},
		{KindInvalidInput, StatusBadRequest},
		{KindPreconditionFailed, StatusPreconditionFailed},
		{KindRequestTimeout, StatusRequestTimeout},
		{KindClientClosedRequest, StatusClientClosedRequest},
		{KindRequestEntityTooLarge, StatusRequestEntityTooLarge},
		{KindServiceUnavailable, StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.kind.HTTPStatus())
		})
	}
}

// TestParseURNRestoresKind ensures a URN that round-trips through the
// wire still resolves to the same HTTP status. Without the registry
// lookup in ParseCode this fails for every code that relies on Kind.
func TestParseURNRestoresKind(t *testing.T) {
	t.Parallel()

	for _, code := range allCodes() {
		got, err := ParseURN(code.URN())
		require.NoError(t, err)
		require.Equal(t, code.Kind, got.Kind, "kind lost for %s", code.URN())
		require.Equal(t, code.HTTPStatus(), got.HTTPStatus(), "status differs for %s", code.URN())
	}
}

func TestCodeHTTPStatus_KnownExceptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code Code
		want HTTPStatus
	}{
		// Driven by Kind:
		{User.BadRequest.RequestTimeout, StatusRequestTimeout},
		{User.BadRequest.ClientClosedRequest, StatusClientClosedRequest},
		{User.BadRequest.RequestBodyTooLarge, StatusRequestEntityTooLarge},
		{App.Validation.InvalidInput, StatusBadRequest},
		{App.Precondition.PreconditionFailed, StatusPreconditionFailed},
		{App.Protection.ProtectedResource, StatusPreconditionFailed},

		// Inherited from category:
		{User.BadRequest.RequestBodyUnreadable, StatusBadRequest},
		{Auth.Authentication.Missing, StatusUnauthorized},
		{Auth.Authentication.Malformed, StatusUnauthorized},

		// CategoryUnkeyData has no default; Kind drives every status.
		{Data.Key.NotFound, StatusNotFound},
		{Data.Workspace.NotFound, StatusNotFound},
		{Data.Api.NotFound, StatusNotFound},
		{Data.Permission.Duplicate, StatusConflict},
		{Data.Role.Duplicate, StatusConflict},
		{Data.Identity.Duplicate, StatusConflict},
		{Data.RatelimitNamespace.Gone, StatusGone},
		{Data.Analytics.NotConfigured, StatusPreconditionFailed},
		{Data.Analytics.ConnectionFailed, StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(string(tt.code.URN()), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.code.HTTPStatus())
		})
	}
}

// allCategories enumerates every Category constant defined in this
// package. Hand-maintained because Go has no reflection over
// package-level constants of a named string type.
func allCategories() []Category {
	return []Category{
		CategoryUserBadRequest,
		CategoryUserUnprocessableEntity,
		CategoryUserTooManyRequests,
		CategoryNotFound,
		CategoryBadGateway,
		CategoryServiceUnavailable,
		CategoryGatewayTimeout,
		CategoryUnauthorized,
		CategoryForbidden,
		CategoryRateLimited,
		CategoryInternalServerError,
		CategoryUnkeyData,
		CategoryUnkeyAuthentication,
		CategoryUnkeyAuthorization,
		CategoryUnkeyLimits,
		CategoryUnkeyApplication,
		CategoryUnkeyVault,
	}
}

// allCodes walks the namespace structs (User, App, Auth, Vault, Data,
// Frontline, Portal, …) and returns every Code value reachable from
// them via reflection. The walker descends into struct fields and only
// collects values of the Code type.
func allCodes() []Code {
	roots := []any{User, App, Auth, Data, Frontline, Portal}
	var out []Code
	codeType := reflect.TypeOf((*Code)(nil)).Elem()
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Struct:
			if v.Type() == codeType {
				out = append(out, v.Interface().(Code))
				return
			}
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i))
			}
		}
	}
	for _, r := range roots {
		walk(reflect.ValueOf(r))
	}
	return out
}
