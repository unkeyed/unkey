package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

// canonicalReadGrants are the canonical grants the keys:read ceiling requires on
// one keyspace, mirroring the list-keys route: key read and keyspace read.
func canonicalReadGrants(workspaceID, projectID, keyspaceID string) []string {
	return []string{
		fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/*#read", workspaceID, projectID, keyspaceID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#read", workspaceID, projectID, keyspaceID),
	}
}

// canonicalRerollGrant is the canonical grant the keys:reroll ceiling requires on
// one keyspace. The key segment is a wildcard because mint time has no key id.
func canonicalRerollGrant(workspaceID, projectID, keyspaceID string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/*#write", workspaceID, projectID, keyspaceID)
}

// TestCreateSessionCeilingAcceptsCanonicalGrants guarantees the mint-time
// ceiling accepts its canonical form on a single keyspace, with the legacy rows
// standing as the regression guard for unchanged behaviour.
func TestCreateSessionCeilingAcceptsCanonicalGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	keyspaceID := api.KeyAuthID.String
	insertKeyspacePortal(t, h, workspaceID, "canonical-ceiling-portal", keyspaceID)

	// A real key id, so the concrete-key case is not a shape the evaluator might
	// treat differently.
	concreteKey := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspaceID,
		KeySpaceID:  keyspaceID,
	})

	canonicalRead := canonicalReadGrants(workspaceID, api.ProjectID, keyspaceID)
	mint := "portal.*.create_portal_session"
	read := []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead}
	reroll := []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead, openapi.KeysReroll}

	testCases := []struct {
		name       string
		scopes     []openapi.V2PortalCreateSessionRequestBodyScopes
		grants     []string
		shouldPass bool
	}{
		{
			name:       "canonical keyspace and key read satisfies read",
			scopes:     read,
			grants:     append([]string{mint}, canonicalRead...),
			shouldPass: true,
		},
		{
			// The canonical arm is itself a conjunction, so either half alone
			// must fail.
			name:       "canonical key read alone does not satisfy read",
			scopes:     read,
			grants:     []string{mint, canonicalRead[0]},
			shouldPass: false,
		},
		{
			name:       "canonical keyspace read alone does not satisfy read",
			scopes:     read,
			grants:     []string{mint, canonicalRead[1]},
			shouldPass: false,
		},
		{
			name:       "legacy tuples still satisfy read",
			scopes:     read,
			grants:     []string{mint, "api.*.read_key", "api.*.read_api"},
			shouldPass: true,
		},
		{
			name:       "canonical key write satisfies reroll on a plaintext keyspace",
			scopes:     reroll,
			grants:     append([]string{mint, canonicalRerollGrant(workspaceID, api.ProjectID, keyspaceID)}, canonicalRead...),
			shouldPass: true,
		},
		{
			// Rerolling from a session reaches every key in the keyspace, so a
			// caller granted one key must not be able to mint it.
			name:       "a grant on one concrete key does not satisfy reroll",
			scopes:     reroll,
			grants:     append([]string{mint, fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#write", workspaceID, api.ProjectID, keyspaceID, concreteKey.KeyID)}, canonicalRead...),
			shouldPass: false,
		},
		{
			// The disjunction is per scope as well as per keyspace.
			name:       "canonical read mixes with a legacy reroll grant",
			scopes:     reroll,
			grants:     append([]string{mint, "api.*.create_key"}, canonicalRead...),
			shouldPass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, tc.grants...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Portal:     "canonical-ceiling-portal",
				ExternalId: "user_canonical_ceiling",
				Scopes:     tc.scopes,
			})
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200, got: %s", res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, got: %s", res.RawBody)
		})
	}
}

// TestCreateSessionCeilingInheritsRerollURNWeakness pins a deliberate weakness:
// the operator reroll route resolves its create and encryption arms to the same
// canonical key-write leaf, so key write alone mints a reroll session on a
// keyspace storing recoverable key material. The ceiling mirrors the route
// rather than being stricter; the legacy form still demands encrypt_key.
func TestCreateSessionCeilingInheritsRerollURNWeakness(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID, EncryptedKeys: true})
	keyspaceID := api.KeyAuthID.String
	insertKeyspacePortal(t, h, workspaceID, "canonical-encrypted-portal", keyspaceID)

	grants := append([]string{
		"portal.*.create_portal_session",
		canonicalRerollGrant(workspaceID, api.ProjectID, keyspaceID),
	}, canonicalReadGrants(workspaceID, api.ProjectID, keyspaceID)...)

	rootKey := h.CreateRootKey(workspaceID, grants...)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     "canonical-encrypted-portal",
		ExternalId: "user_canonical_encrypted",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead, openapi.KeysReroll},
	})
	require.Equal(t, http.StatusOK, res.Status,
		"the canonical arm mirrors the reroll route, whose encryption conjunct is the same key-write leaf, got: %s", res.RawBody)
}

// TestCreateSessionCeilingComposesPerKeyspace guarantees the composition across
// an app-mapped portal's several keyspaces: a conjunction over keyspaces of a
// per-keyspace disjunction between the legacy and canonical forms.
func TestCreateSessionCeilingComposesPerKeyspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspaceID,
		Name:             "ceiling",
		Slug:             "ceiling-" + uid.DNS1035(),
		DeleteProtection: false,
	})
	first := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID, ProjectID: project.ID})
	second := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID, ProjectID: project.ID})

	app := seedAppWithKeyspaces(t, h, workspaceID, "ceiling", project.ID, []string{
		first.KeyAuthID.String,
		second.KeyAuthID.String,
	})
	h.SeedPortal(t, workspaceID, "ceiling-portal", "ceiling-portal", appMapping(app.AppID), nil, nil)

	mint := "portal.*.create_portal_session"
	canonicalFirst := canonicalReadGrants(workspaceID, project.ID, first.KeyAuthID.String)
	canonicalSecond := canonicalReadGrants(workspaceID, project.ID, second.KeyAuthID.String)

	testCases := []struct {
		name       string
		grants     []string
		shouldPass bool
	}{
		{
			// The two forms may be mixed across keyspaces.
			name: "legacy on one keyspace and canonical on the other",
			grants: append([]string{
				mint,
				fmt.Sprintf("api.%s.read_key", first.ID),
				fmt.Sprintf("api.%s.read_api", first.ID),
			}, canonicalSecond...),
			shouldPass: true,
		},
		{
			name:       "canonical on both keyspaces",
			grants:     append(append([]string{mint}, canonicalFirst...), canonicalSecond...),
			shouldPass: true,
		},
		{
			name:       "canonical on only one keyspace",
			grants:     append([]string{mint}, canonicalFirst...),
			shouldPass: false,
		},
		{
			// A complete canonical set on one keyspace plus half a set on the
			// other is short.
			name:       "complete canonical on one keyspace and partial on the other",
			grants:     append(append([]string{mint}, canonicalFirst...), canonicalSecond[0]),
			shouldPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, tc.grants...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Portal:     "ceiling-portal",
				ExternalId: "user_ceiling",
				Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
			})
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200, got: %s", res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, got: %s", res.RawBody)
		})
	}
}
