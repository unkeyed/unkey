package codes

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

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
		// Categories with no canonical default: data resolves per code,
		// vault is non-HTTP.
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

// TestCategoryHTTPStatusCoverage walks every Category constant declared in this
// package and asserts that HTTPStatus() returns either a valid status or 0. A
// new category added without a mapping will fail here by falling through to the
// default branch and returning 0 outside the allowlist.
func TestCategoryHTTPStatusCoverage(t *testing.T) {
	t.Parallel()

	// Categories that legitimately have no canonical HTTP status: vault is
	// non-HTTP (internal infrastructure); data is too broad for a single
	// default and resolves per code via codeStatusOverrides.
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

// TestCodeHTTPStatusCoverage walks every Code declared at package level and
// asserts that each resolves to a non-zero HTTPStatus, either via its category
// default or via codeStatusOverrides. A new data-style code added without an
// override will fail here because CategoryUnkeyData has no default. Vault codes
// are exempt (non-HTTP subsystem).
func TestCodeHTTPStatusCoverage(t *testing.T) {
	t.Parallel()

	for _, code := range allCodes() {
		if code.Category == CategoryUnkeyVault {
			continue
		}
		require.NotEqual(t, HTTPStatus(0), code.HTTPStatus(),
			"code %s resolves to HTTPStatus 0; add a codeStatusOverrides entry or fix the Category",
			code.URN())
	}
}

// TestCodeStatusOverridesNotRedundant guards against override entries that
// merely repeat the category default. A redundant entry means the override map
// and the category taxonomy have drifted and the entry should be deleted: the
// category already produces the right status. (The exception is the data
// category, whose default is 0 by design, so every data override is load-bearing.)
func TestCodeStatusOverridesNotRedundant(t *testing.T) {
	t.Parallel()

	for code, status := range codeStatusOverrides {
		if code.Category == CategoryUnkeyData {
			continue
		}
		require.NotEqual(t, code.Category.HTTPStatus(), status,
			"override for %s duplicates its category default (%d); delete it",
			code.URN(), status.Int())
	}
}

// TestParseURNPreservesStatus ensures a URN that round-trips through the wire
// still resolves to the same HTTP status. Because Code carries no hidden state,
// ParseURN reconstructs an equal Code and the override map resolves identically
// on both sides of an RPC boundary.
func TestParseURNPreservesStatus(t *testing.T) {
	t.Parallel()

	for _, code := range allCodes() {
		got, err := ParseURN(code.URN())
		require.NoError(t, err)
		require.Equal(t, code.HTTPStatus(), got.HTTPStatus(), "status differs for %s", code.URN())
	}
}

func TestCodeHTTPStatus_KnownExceptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code Code
		want HTTPStatus
	}{
		// Driven by a per-code override:
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

		// CategoryUnkeyData has no default; the override map drives every status.
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

// allCategories enumerates every Category constant defined in this package.
// Hand-maintained because Go has no reflection over package-level constants of
// a named string type.
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

// allCodes walks the namespace structs (User, App, Auth, Data, Frontline,
// Portal) and returns every Code value reachable from them. Reflection is
// confined to tests: production resolution is a plain map lookup plus a switch.
func allCodes() []Code {
	roots := []any{User, App, Auth, Data, Frontline, Portal}
	var out []Code
	codeType := reflect.TypeOf((*Code)(nil)).Elem()
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if v.Kind() != reflect.Struct {
			return
		}
		if v.Type() == codeType {
			out = append(out, v.Interface().(Code))
			return
		}
		for i := 0; i < v.NumField(); i++ {
			walk(v.Field(i))
		}
	}
	for _, r := range roots {
		walk(reflect.ValueOf(r))
	}
	return out
}
