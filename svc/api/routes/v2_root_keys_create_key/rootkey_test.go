package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

func TestRootKeyDelegatesCreationThroughBearerAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
		InternalWorkspaceID: r.RootWorkspace.ID,
		InternalKeyspaceID:  r.RootKeySpace.ID,
		InternalProjectID:   r.RootKeySpace.ProjectID,
	}
	h.Register(route)
	canonical := "unkey:v1:" + r.UserWorkspace.ID + ":rootKeys/*#write"
	for _, grant := range []string{canonical, "unkey:v1:" + r.UserWorkspace.ID + ":**#*"} {
		t.Run(grant, func(t *testing.T) {
			bearer := h.CreateRootKey(r.UserWorkspace.ID, grant)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + bearer}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{canonical, canonical}})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			child, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
			require.NoError(t, err)
			require.Equal(t, r.UserWorkspace.ID, child.ForWorkspaceID.String)
			require.Equal(t, r.RootWorkspace.ID, child.WorkspaceID)
			require.False(t, child.IdentityID.Valid)
			grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: child.ID})
			require.NoError(t, err)
			require.Equal(t, []string{canonical}, grants)
			grandchild := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + res.Body.Data.Key}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{canonical}})
			require.Equal(t, http.StatusOK, grandchild.Status, "%s", grandchild.RawBody)
			grandchildGrants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: grandchild.Body.Data.KeyId})
			require.NoError(t, err)
			require.ElementsMatch(t, []string{canonical}, grandchildGrants)
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
		InternalWorkspaceID: r.RootWorkspace.ID,
		InternalKeyspaceID:  r.RootKeySpace.ID,
		InternalProjectID:   r.RootKeySpace.ProjectID,
	}
	h.Register(route)
	base := "unkey:v1:" + r.UserWorkspace.ID + ":"
	canonical := base + "rootKeys/*#write"
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
		{"read is not write", []string{base + "rootKeys/*#read"}, []string{canonical}, 403},
		{"wrong workspace capability", []string{"unkey:v1:" + foreign.WorkspaceID + ":rootKeys/*#write"}, []string{canonical}, 403},
		{"concrete key does not cover creation wildcard", []string{base + "rootKeys/key_one#write"}, []string{canonical}, 403},
		{"concrete subtree does not cover creation wildcard", []string{base + "rootKeys/key_one/**#write"}, []string{canonical}, 403},
		{"project subtree does not cover workspace root keys", []string{base + "projects/*/**#write"}, []string{canonical}, 403},
		{"creation cannot grant canonical global", []string{canonical}, []string{base + "**#*"}, 403},
		{"legacy star cannot grant canonical global", []string{canonical, "*"}, []string{base + "**#*"}, 403},
		{"legacy star cannot grant descendant write", []string{canonical, "*"}, []string{base + "projects/" + api.ProjectID + "/**#write"}, 403},
		{"legacy star cannot grant workspace write", []string{canonical, "*"}, []string{base + "**#write"}, 403},
		{"legacy requests are invalid even for admins", []string{base + "**#*", "*"}, []string{"*"}, 400},
		{"creation cannot grant unrelated action", []string{canonical}, []string{decrypt}, 403},
		{"single key cannot grant all keys", []string{canonical, scope + "/keys/key_one#decrypt"}, []string{decrypt}, 403},
		{"canonical scope cannot expand", []string{canonical, decrypt}, []string{base + "projects/*/keyspaces/*/keys/*#decrypt"}, 403},
		{"foreign requested creation", []string{canonical}, []string{"unkey:v1:" + foreign.WorkspaceID + ":rootKeys/*#write"}, 400},
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
