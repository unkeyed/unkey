package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

func TestRestrictedRootKeyCannotDelegateBroaderNetworks(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	require.NoError(t, db.Query.UpdateRootKeyIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateRootKeyIPAllowlistEnabledParams{WorkspaceID: workspace.ID, Enabled: true}))
	route := &handler.Handler{DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#write"
	admin := h.CreateRootKey(workspace.ID, permission)
	parent := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer " + admin}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{permission}, IpAllowlist: &[]string{"192.0.2.0/25", "2001:db8:1234::/48"}})
	require.Equal(t, http.StatusOK, parent.Status, "%s", parent.RawBody)
	for _, test := range []struct {
		name       string
		allowedIPs *[]string
		status     int
	}{
		{"omitted", nil, http.StatusForbidden},
		{"unrestricted", &[]string{}, http.StatusForbidden},
		{"broader subnet", &[]string{"192.0.2.0/24"}, http.StatusForbidden},
		{"extra network", &[]string{"192.0.2.0/25", "203.0.113.1"}, http.StatusForbidden},
		{"broader IPv6 subnet", &[]string{"2001:db8::/32"}, http.StatusForbidden},
		{"invalid address", &[]string{"192.0.2.999"}, http.StatusBadRequest},
		{"invalid prefix", &[]string{"192.0.2.1/33"}, http.StatusBadRequest},
		{"scoped address", &[]string{"fe80::1%eth0"}, http.StatusBadRequest},
		{"mapped prefix outside IPv4", &[]string{"::ffff:192.0.2.1/80"}, http.StatusBadRequest},
		{"narrower", &[]string{"192.0.2.0/26", "2001:db8:1234:abcd::/64"}, http.StatusOK},
		{"mapped IPv4", &[]string{"::ffff:192.0.2.1"}, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + parent.Body.Data.Key}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{permission}, IpAllowlist: test.allowedIPs})
			require.Equal(t, test.status, res.Status, "%s", res.RawBody)
			if test.status != http.StatusOK {
				require.Equal(t, before, snapshot(t, h))
			}
		})
	}
}

// TestExpiringRootKeyBoundsChildLifetime guarantees a child cannot outlive its
// caller. For example, equal or earlier expiry succeeds, but one millisecond
// later, null, or omitted expiry fails without storing a key or audit events.
func TestExpiringRootKeyBoundsChildLifetime(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
	}
	h.Register(route)
	permission := "unkey:v1:" + r.UserWorkspace.ID + ":rootKeys/*#write"
	expires := h.Clock.Now().Add(time.Hour).Truncate(time.Second)
	parent := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: r.RootWorkspace.ID, KeySpaceID: r.RootKeySpace.ID,
		ForWorkspaceID: new(r.UserWorkspace.ID), Expires: &expires,
		Permissions: []seed.CreatePermissionRequest{{WorkspaceID: r.RootWorkspace.ID, Name: permission, Slug: permission}},
	})
	for _, tt := range []struct {
		name    string
		expires nullable.Nullable[int64]
		status  int
	}{
		{"omitted", nullable.Nullable[int64]{}, http.StatusBadRequest},
		{"null", nullable.NewNullNullable[int64](), http.StatusBadRequest},
		{"later", nullable.NewNullableWithValue(expires.UnixMilli() + 1), http.StatusBadRequest},
		{"equal", nullable.NewNullableWithValue(expires.UnixMilli()), http.StatusOK},
		{"earlier", nullable.NewNullableWithValue(expires.Add(-time.Minute).UnixMilli()), http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + parent.Key}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{permission}, Expires: tt.expires})
			require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
			if tt.status != http.StatusOK {
				require.Equal(t, before, snapshot(t, h))
				return
			}
			child, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
			require.NoError(t, err)
			require.True(t, child.Expires.Valid)
			require.Equal(t, tt.expires.MustGet(), child.Expires.Int64)
		})
	}
}

// TestCLISessionBoundsRootKeyLifetime guarantees a root key minted by a CLI
// session cannot outlive that session. Omitted, null, and later expiry are
// rejected. Equal and earlier expiry are stored.
func TestCLISessionBoundsRootKeyLifetime(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
	}
	expires := h.Clock.Now().Add(time.Hour).Truncate(time.Second)
	permission := "unkey:v1:" + r.UserWorkspace.ID + ":rootKeys/*#write"
	p := &principal.Principal{
		Type: principal.TypeJWT,
		Subject: principal.Subject{
			ID:   "user_cli",
			Type: principal.SubjectTypeUser,
		},
		Source: principal.CLISessionSource{
			SessionID:  "clis_test",
			DeviceName: "laptop",
			ExpiresAt:  expires,
		},
		AuthorizedWorkspaceID: r.UserWorkspace.ID,
		Permissions:           []string{permission},
	}
	middlewares := append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			s.SetPrincipal(p)
			return next(ctx, s)
		}
	})
	h.Register(route, middlewares...)

	for _, tt := range []struct {
		name    string
		expires nullable.Nullable[int64]
		status  int
	}{
		{"omitted", nullable.Nullable[int64]{}, http.StatusBadRequest},
		{"null", nullable.NewNullNullable[int64](), http.StatusBadRequest},
		{"later", nullable.NewNullableWithValue(expires.UnixMilli() + 1), http.StatusBadRequest},
		{"equal", nullable.NewNullableWithValue(expires.UnixMilli()), http.StatusOK},
		{"earlier", nullable.NewNullableWithValue(expires.Add(-time.Minute).UnixMilli()), http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{permission}, Expires: tt.expires})
			require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
			if tt.status != http.StatusOK {
				require.Equal(t, before, snapshot(t, h))
				return
			}
			child, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
			require.NoError(t, err)
			require.True(t, child.Expires.Valid)
			require.Equal(t, tt.expires.MustGet(), child.Expires.Int64)
		})
	}
}

func TestRootKeyDelegatesCreationThroughBearerAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
	}
	h.Register(route)
	permission := "unkey:v1:" + r.UserWorkspace.ID + ":rootKeys/*#write"
	for _, grant := range []string{permission, "unkey:v1:" + r.UserWorkspace.ID + ":**#*"} {
		t.Run(grant, func(t *testing.T) {
			bearer := h.CreateRootKey(r.UserWorkspace.ID, grant)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + bearer}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{permission, permission}})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			child, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
			require.NoError(t, err)
			require.Equal(t, r.UserWorkspace.ID, child.WorkspaceID)
			grants, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{WorkspaceID: r.UserWorkspace.ID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, PrincipalID: child.ID})
			require.NoError(t, err)
			require.Equal(t, []string{permission}, grants)
			grandchild := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + res.Body.Data.Key}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{permission}})
			require.Equal(t, http.StatusOK, grandchild.Status, "%s", grandchild.RawBody)
			grandchildGrants, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{WorkspaceID: r.UserWorkspace.ID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, PrincipalID: grandchild.Body.Data.KeyId})
			require.NoError(t, err)
			require.ElementsMatch(t, []string{permission}, grandchildGrants)
			logs := h.FindAuditLogsByTargetID(t.Context(), t, grandchild.Body.Data.KeyId)
			require.Len(t, logs, 2)
			for _, log := range logs {
				require.Equal(t, "rootkey", log.Actor.Type)
				require.Equal(t, child.ID, log.Actor.ID)
				require.Equal(t, r.UserWorkspace.ID, log.WorkspaceID)
				require.NotEmpty(t, log.CorrelationID)
				require.Equal(t, logs[0].CorrelationID, log.CorrelationID)
			}
			before := snapshot(t, h)
			escalation := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + res.Body.Data.Key}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{"unkey:v1:" + r.UserWorkspace.ID + ":**#*"}})
			require.Equal(t, http.StatusForbidden, escalation.Status, "%s", escalation.RawBody)
			require.Equal(t, before, snapshot(t, h))
		})
	}
}

func TestRootKeyRejectsUnauthorizedCreationWithoutWrites(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
	}
	h.Register(route)
	base := "unkey:v1:" + r.UserWorkspace.ID + ":"
	permission := base + "rootKeys/*#write"
	foreign := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.CreateWorkspace().ID})
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: r.UserWorkspace.ID})
	scope := base + "projects/" + api.ProjectID + "/keyspaces/" + api.KeyAuthID.String
	decrypt := scope + "/keys/*#decrypt"
	for _, tt := range []struct {
		name              string
		caller, requested []string
		status            int
	}{
		{"no creation capability", []string{decrypt}, []string{decrypt}, 403},
		{"read is not write", []string{base + "rootKeys/*#read"}, []string{permission}, 403},
		{"wrong workspace capability", []string{"unkey:v1:" + foreign.WorkspaceID + ":rootKeys/*#write"}, []string{permission}, 403},
		{"concrete key does not cover creation wildcard", []string{base + "rootKeys/key_one#write"}, []string{permission}, 403},
		{"concrete subtree does not cover creation wildcard", []string{base + "rootKeys/key_one/**#write"}, []string{permission}, 403},
		{"project subtree does not cover workspace root keys", []string{base + "projects/*/**#write"}, []string{permission}, 403},
		{"creation cannot grant global access", []string{permission}, []string{base + "**#*"}, 403},
		{"legacy star cannot grant global access", []string{permission, "*"}, []string{base + "**#*"}, 403},
		{"legacy star cannot grant descendant write", []string{permission, "*"}, []string{base + "projects/" + api.ProjectID + "/**#write"}, 403},
		{"legacy star cannot grant workspace write", []string{permission, "*"}, []string{base + "**#write"}, 403},
		{"legacy requests are invalid even for admins", []string{base + "**#*", "*"}, []string{"*"}, 400},
		{"creation cannot grant unrelated action", []string{permission}, []string{decrypt}, 403},
		{"single key cannot grant all keys", []string{permission, scope + "/keys/key_one#decrypt"}, []string{decrypt}, 403},
		{"permission scope cannot expand", []string{permission, decrypt}, []string{base + "projects/*/keyspaces/*/keys/*#decrypt"}, 403},
		{"foreign requested creation", []string{permission}, []string{"unkey:v1:" + foreign.WorkspaceID + ":rootKeys/*#write"}, 400},
		{"foreign legacy API", []string{base + "**#*"}, []string{"api." + foreign.ID + ".decrypt_key"}, 400},
		{"no workspace ID legacy equivalence", []string{base + "**#*"}, []string{"workspace." + r.UserWorkspace.ID + ".create_root_key"}, 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bearer := h.CreateRootKey(r.UserWorkspace.ID, tt.caller...)
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + bearer}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: tt.requested})
			require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
			require.Equal(t, before, snapshot(t, h))
		})
	}
}
