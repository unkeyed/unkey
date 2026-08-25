package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

// exchangeCodeFromURL extracts the single-use code the portal URL carries.
func exchangeCodeFromURL(t *testing.T, portalURL string) string {
	t.Helper()

	parsed, err := url.Parse(portalURL)
	require.NoError(t, err)

	code := parsed.Query().Get("code")
	require.NotEmpty(t, code)
	return code
}

func TestCreateSessionSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	// A keyspace-mapped portal: the session is scoped to this keyspace, derived
	// from the portal rather than the request. The keyspace is created through
	// CreateApi so it has an owning api, which is what production always has and
	// what the mint-time ceiling needs to express its api-scoped checks.
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspaceID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	keySpaceID := api.KeyAuthID.String

	portalID := insertKeyspacePortal(t, h, workspaceID, "test-portal", keySpaceID)

	// Every subtest below requests keys:read, so the caller needs the portal
	// permission plus the read-keys conjunction the equivalent operator route
	// demands. Without the second conjunct the mint is refused at stage 2.
	rootKey := h.CreateRootKey(workspaceID,
		"portal.*.create_portal_session",
		"api.*.read_key",
		"api.*.read_api",
	)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("basic session creation", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_123",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.Id)
		require.NotEmpty(t, res.Body.Data.Url)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// The response returns the non-secret handle; the credential lives only
		// in the URL, so the id must not appear there.
		require.True(t, strings.HasPrefix(res.Body.Data.Id, "ps_"))
		require.Contains(t, res.Body.Data.Url, "portal.unkey.com")
		require.NotContains(t, res.Body.Data.Url, res.Body.Data.Id)
		require.True(t, strings.HasPrefix(res.Body.Data.Url, "https://"))
	})

	t.Run("resolves the portal by id as well as slug", func(t *testing.T) {
		req := handler.Request{
			Portal:     portalID,
			ExternalId: "user_by_id",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotEmpty(t, res.Body.Data.Id)
	})

	t.Run("with preview", func(t *testing.T) {
		preview := true
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_789",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
			Preview:    &preview,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.Id)

		// The session persists in the pending state: the code's hash is stored,
		// no access token has been issued.
		code := exchangeCodeFromURL(t, res.Body.Data.Url)
		session, err := db.Query.FindPortalSessionByExchangeCodeHash(ctx, h.DB.RO(), hash.Sha256(code))
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.Id, session.ID)
		require.Equal(t, "user_789", session.ExternalID)
		require.Equal(t, portalID, session.PortalID)
		require.True(t, session.Preview)
		require.False(t, session.AccessTokenHash.Valid)
	})

	t.Run("stores the exchange code only as a hash", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_hashed",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		code := exchangeCodeFromURL(t, res.Body.Data.Url)

		// The plaintext code must not be queryable: only its hash resolves.
		_, err := db.Query.FindPortalSessionByExchangeCodeHash(ctx, h.DB.RO(), code)
		require.True(t, db.IsNotFound(err), "plaintext exchange code must not be stored")

		session, err := db.Query.FindPortalSessionByExchangeCodeHash(ctx, h.DB.RO(), hash.Sha256(code))
		require.NoError(t, err)
		require.NotEqual(t, code, session.ExchangeCodeHash)
	})

	t.Run("multiple sessions for same externalId", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_multi",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)

		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status)

		// Each call must produce a unique session and a unique code.
		require.NotEqual(t, res1.Body.Data.Id, res2.Body.Data.Id)
		require.NotEqual(t, res1.Body.Data.Url, res2.Body.Data.Url)
	})
}

// TestCreateSessionAuditsGrantedScopesAndKeyspaces asserts that the audit
// entry for a created session records what the session was actually granted.
// Without it, the log says a session was minted but not what it can do, which
// is the question an incident responder asks first.
//
// Asserted here rather than in the authorization tests because it is a side
// effect of the success path -- a denied request writes no entry at all, which
// 403_test.go already pins.
func TestCreateSessionAuditsGrantedScopesAndKeyspaces(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspaceID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	insertKeyspacePortal(t, h, workspaceID, "audit-portal", api.KeyAuthID.String)

	// Several scopes, so the assertion pins the whole granted set rather than
	// passing on a single-element slice.
	rootKey := h.CreateRootKey(workspaceID,
		"portal.*.create_portal_session",
		"api.*.read_key",
		"api.*.read_api",
		"api.*.create_key",
		"api.*.read_analytics",
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     "audit-portal",
		ExternalId: "user_audit",
		Scopes: []openapi.V2PortalCreateSessionRequestBodyScopes{
			openapi.KeysRead, openapi.KeysReroll, openapi.AnalyticsRead,
		},
	})
	require.Equal(t, 200, res.Status, "got: %s", res.RawBody)
	sessionID := res.Body.Data.Id
	require.NotEmpty(t, sessionID)

	events := h.FindAuditLogsByTargetID(ctx, t, sessionID)
	require.Len(t, events, 1, "exactly one audit entry per minted session")

	var target *auditlog.EventTarget
	for i := range events[0].Targets {
		if events[0].Targets[i].ID == sessionID {
			target = &events[0].Targets[i]
			break
		}
	}
	require.NotNil(t, target, "the session must be a target of its own audit entry")

	// JSON round-trips these as []any, so compare element-wise rather than
	// against a []string literal.
	require.ElementsMatch(t,
		[]any{"keys:read", "keys:reroll", "analytics:read"},
		target.Meta["scopes"],
		"the audit entry must record every granted scope",
	)
	require.ElementsMatch(t,
		[]any{api.KeyAuthID.String},
		target.Meta["keyspaceIds"],
		"the audit entry must record the keyspaces the session is scoped to",
	)

	// Pre-existing metadata must survive alongside the new fields.
	require.Equal(t, "audit-portal", target.Meta["slug"])
	require.NotEmpty(t, target.Meta["portalId"])
}
