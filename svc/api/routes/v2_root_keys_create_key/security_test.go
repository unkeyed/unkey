package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

func TestCreateRejectsBroaderOrInvalidGrantsWithoutWrites(t *testing.T) {
	h, route, p := newHarness(t)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: p.AuthorizedWorkspaceID})
	foreign := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.CreateWorkspace().ID})
	scope := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + api.ProjectID + "/keyspaces/" + api.KeyAuthID.String
	grant := scope + "/keys/*#decrypt"
	global := "unkey:v1:" + p.AuthorizedWorkspaceID + ":**#*"
	allKeys := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/*/keyspaces/*/keys/*#decrypt"
	for _, tt := range []struct {
		name      string
		caller    []string
		requested []string
		status    int
	}{
		{"single key cannot grant all keys", []string{scope + "/keys/key_one#decrypt"}, []string{grant}, 403},
		{"one keyspace cannot grant all keyspaces", []string{grant}, []string{allKeys}, 403},
		{"project wildcard cannot grant all projects", []string{"unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + api.ProjectID + "/**#decrypt"}, []string{allKeys}, 403},
		{"narrow grants cannot grant global", []string{grant}, []string{global}, 403},
		{"legacy star cannot grant portal sessions", []string{"*"}, []string{"unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + api.ProjectID + "/portals/portal_one/sessions/*#write"}, 403},
		{"action must be contained", []string{scope + "/keys/*#read"}, []string{grant}, 403},
		{"foreign caller scope is rejected", []string{strings.Replace(grant, p.AuthorizedWorkspaceID, foreign.WorkspaceID, 1)}, []string{grant}, 403},
		{"foreign URN is rejected", []string{global}, []string{strings.Replace(grant, p.AuthorizedWorkspaceID, foreign.WorkspaceID, 1)}, 400},
		{"multiple action separators", []string{global}, []string{grant + "#delete"}, 400},
		{"partial wildcard is not a scope", []string{global}, []string{scope + "/keys/key_*#decrypt"}, 400},
		{"fullwidth path cannot become a descendant grant", []string{"unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/*/keyspaces/*#write"}, []string{scope + "／keys／＊#write"}, 400},
		{"one unsupported grant rejects whole request", []string{global}, []string{grant, scope + "#rotate"}, 400},
		{"null permissions", []string{global}, nil, 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p.Permissions = append([]string{"unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write"}, tt.caller...)
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{Permissions: tt.requested})
			require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
			require.Equal(t, before, snapshot(t, h))
		})
	}
}

func TestCreateAcceptsContainedDescendants(t *testing.T) {
	h, route, p := newHarness(t)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: p.AuthorizedWorkspaceID})
	base := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + api.ProjectID
	for _, caller := range []string{base + "/**#decrypt", base + "/keyspaces/*/keys/*#decrypt", base + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#decrypt"} {
		t.Run(caller, func(t *testing.T) {
			p.Permissions = []string{caller, "unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write"}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
				Permissions: []string{base + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#decrypt"},
				Expires:     nullable.NewNullNullable[int64](),
			})
			require.Equal(t, http.StatusOK, res.Status)
			key, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
			require.NoError(t, err)
			require.False(t, key.Expires.Valid)
		})
	}
}

func TestPermissionProjectConflictRollsBackKeyGrantsAndAudit(t *testing.T) {
	h, route, p := newHarness(t)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: p.AuthorizedWorkspaceID})
	otherProject := h.CreateProject(seed.CreateProjectRequest{WorkspaceID: route.InternalWorkspaceID, ID: uid.New(uid.ProjectPrefix)})
	grant := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + api.ProjectID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#delete"
	require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: uid.New(uid.PermissionPrefix), WorkspaceID: route.InternalWorkspaceID,
		ProjectID: otherProject.ID, Name: grant, Slug: grant,
	}))
	before := snapshot(t, h)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{Permissions: []string{grant}})
	require.Equal(t, http.StatusInternalServerError, res.Status)
	require.Equal(t, before, snapshot(t, h))
}

func TestPermissionCollationSubstitutionRollsBack(t *testing.T) {
	h, route, p := newHarness(t)
	projectID := uid.New(uid.ProjectPrefix)
	permission := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + projectID + "/keyspaces/ks_one#write"
	stored := strings.Replace(permission, "ks_one", "ks_one\u200b", 1)
	require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: uid.New(uid.PermissionPrefix), WorkspaceID: route.InternalWorkspaceID,
		ProjectID: route.InternalProjectID, Name: stored, Slug: stored,
	}))
	var equal bool
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT slug = ? FROM permissions WHERE workspace_id = ? AND slug = ?", permission, route.InternalWorkspaceID, stored).Scan(&equal))
	require.True(t, equal, "fixture must reproduce collation-equivalent but byte-distinct slugs")
	p.Permissions = []string{"unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write", permission}
	before := snapshot(t, h)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{Permissions: []string{permission}})
	require.Equal(t, http.StatusInternalServerError, res.Status, "%s", res.RawBody)
	require.Equal(t, before, snapshot(t, h))
}

func TestPermissionStoragePreservesCase(t *testing.T) {
	h, route, p := newHarness(t)
	base := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/proj_one/keyspaces/"
	requested := []string{base + "ks_one#read", base + "KS_one#read"}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{Permissions: requested})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	stored, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
	require.NoError(t, err)
	require.ElementsMatch(t, requested, stored)
}

func snapshot(t *testing.T, h *testutil.Harness) []int {
	t.Helper()
	var counts []int
	for _, query := range []struct{ sql, workspaceID string }{
		{"SELECT COUNT(*) FROM `keys` WHERE workspace_id = ?", h.Resources().RootWorkspace.ID},
		{"SELECT COUNT(*) FROM permissions WHERE workspace_id = ?", h.Resources().RootWorkspace.ID},
		{"SELECT COUNT(*) FROM keys_permissions WHERE workspace_id = ?", h.Resources().RootWorkspace.ID},
		{"SELECT COUNT(*) FROM clickhouse_outbox WHERE workspace_id = ?", h.Resources().UserWorkspace.ID},
	} {
		var count int
		require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), query.sql, query.workspaceID).Scan(&count))
		counts = append(counts, count)
	}
	return counts
}

func TestAuditDatabaseFailureRollsBackKeyAndGrants(t *testing.T) {
	h, route, p := newHarness(t)
	route.Auditlogs = auditWithFailingDatabase{inner: h.Auditlogs}
	before := snapshot(t, h)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{Permissions: p.Permissions})
	require.Equal(t, http.StatusInternalServerError, res.Status)
	require.Equal(t, before, snapshot(t, h))
}

type auditWithFailingDatabase struct {
	inner auditlogs.AuditLogService
}

func (a auditWithFailingDatabase) Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
	return a.inner.Insert(ctx, failedWrites{DBTX: tx}, logs)
}

type failedWrites struct {
	db.DBTX
}

func (failedWrites) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, errors.New("injected audit database write failure")
}
