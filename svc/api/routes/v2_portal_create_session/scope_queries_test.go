package handler_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

// ScopeQueries is unit-tested rather than driven through the route because the
// request enum rejects an unknown scope at the boundary, so the deny-by-default
// arm is unreachable over HTTP. It still has to be pinned: rbac.And over zero
// children evaluates to valid, so a scope that were silently skipped instead of
// denied would mint a session with no check at all.
//
// This file carries no HTTP status in its name on purpose -- the route's other
// test files are named for the status they exercise, and this one exercises none.
func TestScopeQueriesDeniesUnmappedScope(t *testing.T) {
	const apiID = "api_test"

	t.Run("known scopes map to a non-empty requirement", func(t *testing.T) {
		for _, s := range []openapi.V2PortalCreateSessionRequestBodyScopes{
			openapi.KeysRead, openapi.KeysCreate, openapi.KeysReroll, openapi.AnalyticsRead,
		} {
			queries, ok := handler.ScopeQueries(s, apiID, false)
			require.True(t, ok, "scope %q must map", s)
			require.NotEmpty(t, queries, "scope %q must produce at least one check", s)
		}
	})

	t.Run("unknown scope denies", func(t *testing.T) {
		queries, ok := handler.ScopeQueries("keys:destroy", apiID, false)
		require.False(t, ok, "an unmapped scope must deny, not be skipped")
		require.Empty(t, queries)
	})

	t.Run("encryption adds a conjunct for key minting scopes", func(t *testing.T) {
		plain, ok := handler.ScopeQueries(openapi.KeysReroll, apiID, false)
		require.True(t, ok)
		encrypted, ok := handler.ScopeQueries(openapi.KeysReroll, apiID, true)
		require.True(t, ok)
		require.Len(t, plain, 1)
		require.Len(t, encrypted, 2)
	})
}
