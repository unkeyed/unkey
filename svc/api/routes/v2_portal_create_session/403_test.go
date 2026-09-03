package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

// insertKeyspacePortal seeds an enabled portal mapped to a single keyspace.
func insertKeyspacePortal(t *testing.T, h *testutil.Harness, workspaceID, slug, keyspaceID string) string {
	t.Helper()

	projectID := ""
	keyspace, err := db.Query.FindKeySpaceByID(context.Background(), h.DB.RO(), keyspaceID)
	if err == nil {
		projectID = keyspace.ProjectID
	} else {
		require.True(t, db.IsNotFound(err))
	}

	return h.CreatePortal(seed.CreatePortalRequest{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Slug:        slug,
		KeyAuthID:   sql.NullString{Valid: true, String: keyspaceID},
		Enabled:     true,
	}).ID
}

// countPortalSessions counts the sessions minted for one external id. The
// exchange code is the only handle a successful call returns, so a rejected
// call can only be shown to have written nothing by counting rows directly.
func countPortalSessions(t *testing.T, h *testutil.Harness, workspaceID, externalID string) int {
	t.Helper()

	var count int
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM portal_sessions WHERE workspace_id = ? AND external_id = ?",
		workspaceID, externalID,
	).Scan(&count))
	return count
}

// countAuditEntriesMentioning counts outbox audit payloads referencing a string.
func countAuditEntriesMentioning(t *testing.T, h *testutil.Harness, workspaceID, needle string) int {
	t.Helper()

	rows, err := db.Query.ListClickhouseOutboxByWorkspace(context.Background(), h.DB.RO(), workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		if strings.Contains(string(row.Payload), needle) {
			count++
		}
	}
	return count
}

// TestCreateSessionAuthorizationMatrix covers stage 1: whether the caller may
// mint a session for this portal at all.
func TestCreateSessionAuthorizationMatrix(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	portalID := insertKeyspacePortal(t, h, workspace.ID, "matrix-portal", api.KeyAuthID.String)
	otherPortalID := uid.New(uid.PortalPrefix)

	// The keys:read scope requires these of the caller on the portal's api, so
	// every stage-1 pass case carries them: stage 1 is what is under test.
	keyGrants := []string{"api.*.read_key", "api.*.read_api"}

	testCases := []struct {
		name        string
		permissions []string
		portal      string
		shouldPass  bool
	}{
		{name: "no permissions", permissions: []string{}, portal: "matrix-portal", shouldPass: false},
		{name: "wildcard portal permission", permissions: append([]string{"portal.*.create_portal_session"}, keyGrants...), portal: "matrix-portal", shouldPass: true},
		{name: "specific portal permission", permissions: append([]string{fmt.Sprintf("portal.%s.create_portal_session", portalID)}, keyGrants...), portal: "matrix-portal", shouldPass: true},
		{name: "permission and more", permissions: append([]string{"some.other.permission", "portal.*.create_portal_session"}, keyGrants...), portal: "matrix-portal", shouldPass: true},
		// The tuple is built from portal.ID, so naming the portal by slug must
		// match an id-scoped grant.
		{name: "portal named by slug matches id scoped grant", permissions: append([]string{fmt.Sprintf("portal.%s.create_portal_session", portalID)}, keyGrants...), portal: "matrix-portal", shouldPass: true},
		{name: "portal named by id matches id scoped grant", permissions: append([]string{fmt.Sprintf("portal.%s.create_portal_session", portalID)}, keyGrants...), portal: portalID, shouldPass: true},
		{name: "another portals grant", permissions: append([]string{fmt.Sprintf("portal.%s.create_portal_session", otherPortalID)}, keyGrants...), portal: "matrix-portal", shouldPass: false},
		{name: "read portal does not grant minting", permissions: append([]string{"portal.*.read_portal"}, keyGrants...), portal: "matrix-portal", shouldPass: false},
		{name: "update portal does not grant minting", permissions: append([]string{"portal.*.update_portal"}, keyGrants...), portal: "matrix-portal", shouldPass: false},
		{name: "create portal does not grant minting", permissions: append([]string{"portal.*.create_portal"}, keyGrants...), portal: "matrix-portal", shouldPass: false},
		{name: "delete portal does not grant minting", permissions: append([]string{"portal.*.delete_portal"}, keyGrants...), portal: "matrix-portal", shouldPass: false},
		{name: "unrelated api permission", permissions: []string{"api.*.read_api"}, portal: "matrix-portal", shouldPass: false},
		{name: "key permissions alone", permissions: keyGrants, portal: "matrix-portal", shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Portal:     tc.portal,
				ExternalId: "user_matrix",
				Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
			})
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}

			// A stage-1 denial is masked as 404: a caller short of the minting
			// permission must not be able to tell an existing portal from an
			// absent one, and must not learn the resolved portal id. req.Portal
			// accepts a slug, so that id is otherwise unobtainable.
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, got: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, portalID, "a denial must not disclose the resolved portal id")
			require.NotContains(t, res.RawBody, "create_portal_session", "a denial must not echo the required permission")
		})
	}
}

// TestCreateSessionScopeEscalation covers stage 2: a minted session may never
// carry a capability the calling root key does not itself hold.
func TestCreateSessionScopeEscalation(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	plainAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	insertKeyspacePortal(t, h, workspace.ID, "escalation-portal", plainAPI.KeyAuthID.String)

	encryptedAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID, EncryptedKeys: true})
	insertKeyspacePortal(t, h, workspace.ID, "encrypted-portal", encryptedAPI.KeyAuthID.String)

	mint := "portal.*.create_portal_session"

	testCases := []struct {
		name        string
		portal      string
		scopes      []openapi.V2PortalCreateSessionRequestBodyScopes
		permissions []string
		shouldPass  bool
	}{
		{
			// The escalation regression: minting alone must not confer key
			// rotation, which hands back plaintext key material.
			name:        "reroll without create_key",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:reroll"},
			permissions: []string{mint},
			shouldPass:  false,
		},
		{
			name:        "read without key permissions",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
			permissions: []string{mint},
			shouldPass:  false,
		},
		{
			name:        "analytics without read_analytics",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"analytics:read"},
			permissions: []string{mint},
			shouldPass:  false,
		},
		{
			name:        "create without create_key",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:create"},
			permissions: []string{mint},
			shouldPass:  false,
		},
		{
			name:        "read with read_key and read_api",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
			permissions: []string{mint, "api.*.read_key", "api.*.read_api"},
			shouldPass:  true,
		},
		{
			// The read-keys requirement is a conjunction: read_key alone is
			// strictly weaker than the operator route.
			name:        "read with read_key but not read_api",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
			permissions: []string{mint, "api.*.read_key"},
			shouldPass:  false,
		},
		{
			name:        "read with read_api but not read_key",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
			permissions: []string{mint, "api.*.read_api"},
			shouldPass:  false,
		},
		{
			// Rerolling is a create, matching the operator reroll route, not an
			// update.
			name:        "reroll with create_key",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:reroll"},
			permissions: []string{mint, "api.*.create_key"},
			shouldPass:  true,
		},
		{
			name:        "reroll with update_key only",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:reroll"},
			permissions: []string{mint, "api.*.update_key"},
			shouldPass:  false,
		},
		{
			name:        "api scoped grants satisfy the check",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
			permissions: []string{mint, fmt.Sprintf("api.%s.read_key", plainAPI.ID), fmt.Sprintf("api.%s.read_api", plainAPI.ID)},
			shouldPass:  true,
		},
		{
			name:        "analytics with api scoped read_analytics",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"analytics:read"},
			permissions: []string{mint, fmt.Sprintf("api.%s.read_analytics", plainAPI.ID)},
			shouldPass:  true,
		},
		{
			name:        "analytics with wildcard read_analytics",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"analytics:read"},
			permissions: []string{mint, "api.*.read_analytics"},
			shouldPass:  true,
		},
		{
			// Every requested scope must be held: the caller is refused rather
			// than handed the intersection.
			name:        "read granted but reroll not",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read", "keys:reroll"},
			permissions: []string{mint, "api.*.read_key", "api.*.read_api"},
			shouldPass:  false,
		},
		{
			name:        "all requested scopes granted",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read", "keys:reroll", "analytics:read"},
			permissions: []string{mint, "api.*.read_key", "api.*.read_api", "api.*.create_key", "api.*.read_analytics"},
			shouldPass:  true,
		},
		{
			// store_encrypted_keys false: the encrypt_key conjunct must not
			// apply, so create_key alone is enough.
			name:        "reroll on unencrypted keyspace needs only create_key",
			portal:      "escalation-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:reroll"},
			permissions: []string{mint, "api.*.create_key"},
			shouldPass:  true,
		},
		{
			// store_encrypted_keys set: rerolling recovers key material, so
			// encrypt_key is additionally required.
			name:        "reroll on encrypted keyspace without encrypt_key",
			portal:      "encrypted-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:reroll"},
			permissions: []string{mint, "api.*.create_key"},
			shouldPass:  false,
		},
		{
			name:        "reroll on encrypted keyspace with encrypt_key",
			portal:      "encrypted-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:reroll"},
			permissions: []string{mint, "api.*.create_key", "api.*.encrypt_key"},
			shouldPass:  true,
		},
		{
			name:        "create on encrypted keyspace without encrypt_key",
			portal:      "encrypted-portal",
			scopes:      []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:create"},
			permissions: []string{mint, "api.*.create_key"},
			shouldPass:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Portal:     tc.portal,
				ExternalId: "user_escalation",
				Scopes:     tc.scopes,
			})
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200 for %v %v, got: %s", tc.scopes, tc.permissions, res.RawBody)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v %v, got: %s", tc.scopes, tc.permissions, res.RawBody)
			}
		})
	}
}

// TestCreateSessionRejectionWritesNothing guarantees a stage-2 rejection has no
// side effects: no session row, no audit entry.
func TestCreateSessionRejectionWritesNothing(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	insertKeyspacePortal(t, h, workspace.ID, "no-write-portal", api.KeyAuthID.String)

	externalID := "user_no_write_" + uid.New(uid.TestPrefix)
	rootKey := h.CreateRootKey(workspace.ID, "portal.*.create_portal_session", "api.*.read_key", "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     "no-write-portal",
		ExternalId: externalID,
		// keys:reroll is not held, so the whole request is rejected.
		Scopes: []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read", "keys:reroll"},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "got: %s", res.RawBody)

	require.Equal(t, 0, countPortalSessions(t, h, workspace.ID, externalID), "a rejected request must not insert a session")
	require.Equal(t, 0, countAuditEntriesMentioning(t, h, workspace.ID, externalID), "a rejected request must not write an audit log")
}

// TestCreateSessionMultiKeyspacePartialGrant guarantees the requirement holds on
// *every* resolved keyspace, not just one of them.
func TestCreateSessionMultiKeyspacePartialGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	granted := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	ungranted := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	// A deployed app maps to exactly one keyspace today, so the multi-keyspace
	// shape is constructed directly rather than through app provisioning.
	app := seedAppWithKeyspaces(t, h, workspace.ID, granted.ProjectID, "multi-keyspace", []string{
		granted.KeyAuthID.String,
		ungranted.KeyAuthID.String,
	})

	require.NoError(t, db.Query.InsertPortal(ctx, h.DB.RW(), db.InsertPortalParams{
		ID:          uid.New(uid.PortalPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   app.ProjectID,
		Slug:        "multi-keyspace-portal",
		AppID:       sql.NullString{Valid: true, String: app.ID},
		Enabled:     true,
		CreatedAt:   time.Now().UnixMilli(),
	}))

	rootKey := h.CreateRootKey(workspace.ID,
		"portal.*.create_portal_session",
		fmt.Sprintf("api.%s.read_key", granted.ID),
		fmt.Sprintf("api.%s.read_api", granted.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     "multi-keyspace-portal",
		ExternalId: "user_multi_keyspace",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "a grant on one of two keyspaces must not mint, got: %s", res.RawBody)
}

// TestCreateSessionRejectsCrossProjectKeyspace guarantees every keyspace
// resolved through an app belongs to the portal's project.
func TestCreateSessionRejectsCrossProjectKeyspace(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	inProject := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "other project",
		Slug:        "other-project-" + uid.DNS1035(),
	})
	other := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID, ProjectID: otherProject.ID})
	app := seedAppWithKeyspaces(t, h, workspace.ID, inProject.ProjectID, "cross-project", []string{
		inProject.KeyAuthID.String,
		other.KeyAuthID.String,
	})
	stored := h.CreatePortal(seed.CreatePortalRequest{
		WorkspaceID: workspace.ID,
		ProjectID:   inProject.ProjectID,
		Slug:        "cross-project-portal",
		AppID:       sql.NullString{Valid: true, String: app.ID},
		Enabled:     true,
	})

	rootKey := h.CreateRootKey(workspace.ID,
		"portal.*.create_portal_session",
		fmt.Sprintf("api.%s.read_key", inProject.ID),
		fmt.Sprintf("api.%s.read_api", inProject.ID),
		fmt.Sprintf("api.%s.read_key", other.ID),
		fmt.Sprintf("api.%s.read_api", other.ID),
	)
	externalID := "user_cross_project"
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{
		Portal:     stored.ID,
		ExternalId: externalID,
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "a portal must not grant access across projects: %s", res.RawBody)
	require.Equal(t, 0, countPortalSessions(t, h, workspace.ID, externalID), "a rejected request must not insert a session")
}

// TestCreateSessionForbiddenDisabledPortal guarantees the disabled check is
// reached with the permission granted, so it is not shadowed by authorization.
func TestCreateSessionForbiddenDisabledPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	require.NoError(t, db.Query.InsertPortal(ctx, h.DB.RW(), db.InsertPortalParams{
		ID:          uid.New(uid.PortalPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   api.ProjectID,
		Slug:        "disabled-portal",
		KeyAuthID:   sql.NullString{Valid: true, String: api.KeyAuthID.String},
		Enabled:     false,
		CreatedAt:   time.Now().UnixMilli(),
	}))

	rootKey := h.CreateRootKey(workspace.ID,
		"portal.*.create_portal_session",
		"api.*.read_key",
		"api.*.read_api",
		"api.*.create_key",
		"api.*.encrypt_key",
		"api.*.read_analytics",
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{
		Portal:     "disabled-portal",
		ExternalId: "user_123",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	})
	require.Equal(t, http.StatusForbidden, res.Status)
	require.NotNil(t, res.Body)
	require.Equal(t, "Portal is disabled.", res.Body.Error.Detail, "the disabled check must not be shadowed by authorization")
}

// TestCreateSessionKeyspaceWithoutAPI guarantees a keyspace with no owning api
// fails loudly instead of being skipped: every stage-2 requirement is
// api-scoped, so such a keyspace admits no expressible check.
func TestCreateSessionKeyspaceWithoutAPI(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// A keyspace with no api row pointing at it.
	orphanKeyspaceID := uid.New(uid.KeySpacePrefix)
	require.NoError(t, db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            orphanKeyspaceID,
		WorkspaceID:   workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	}))
	insertKeyspacePortal(t, h, workspace.ID, "orphan-portal", orphanKeyspaceID)

	externalID := "user_orphan_" + uid.New(uid.TestPrefix)
	rootKey := h.CreateRootKey(workspace.ID, "portal.*.create_portal_session", "api.*.read_key", "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers, handler.Request{
		Portal:     "orphan-portal",
		ExternalId: externalID,
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "got: %s", res.RawBody)
	require.Equal(t, 0, countPortalSessions(t, h, workspace.ID, externalID))
	require.Equal(t, 0, countAuditEntriesMentioning(t, h, workspace.ID, externalID))
}
