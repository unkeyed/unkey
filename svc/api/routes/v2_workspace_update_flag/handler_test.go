package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_update_flag"
)

func TestUpdateFlagSavesFalse(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	other := h.CreateWorkspace()
	slug := "preview_" + uid.DNS1035()
	require.NoError(t, db.Query.UpsertFlag(t.Context(), h.DB.RW(), db.UpsertFlagParams{
		Slug:         slug,
		Description:  "Uncatalogued preview",
		DefaultValue: true,
		AllowEnable:  true,
		AllowDisable: true,
	}))
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}, handler.Request{Slug: slug, Value: false})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.V2WorkspaceUpdateFlagResponseData{
		Slug:         slug,
		Value:        false,
		Description:  "Uncatalogued preview",
		AllowEnable:  true,
		AllowDisable: true,
	}, res.Body.Data)
	assertOverride(t, h, workspaceID, slug, true, false)
	assertOverride(t, h, other.ID, slug, false, false)
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, flagHeaders(key), handler.Request{Slug: slug, Value: true})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.True(t, res.Body.Data.Value)
	assertOverride(t, h, workspaceID, slug, true, true)
	rows, err := db.Query.ListClickhouseOutboxByWorkspace(t.Context(), h.DB.RW(), workspaceID)
	require.NoError(t, err)
	changes := []map[string]any{}
	for _, row := range rows {
		var event auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &event))
		if event.Event == string(auditlog.WorkspaceUpdateEvent) {
			require.Equal(t, workspaceID, event.WorkspaceID)
			require.Len(t, event.Targets, 1)
			require.Equal(t, workspaceID, event.Targets[0].ID)
			changes = append(changes, event.Targets[0].Meta)
			require.NotEmpty(t, event.Actor.ID)
		}
	}
	require.ElementsMatch(t, []map[string]any{
		{"slug": slug, "before": true, "after": false},
		{"slug": slug, "before": false, "after": true},
	}, changes)
}

func TestUpdateFlagDirectionPolicy(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":flags#write")
	for _, tc := range []struct {
		name                             string
		value, allowEnable, allowDisable bool
		status                           int
	}{
		{"enable allowed", true, true, false, http.StatusOK},
		{"disable allowed", false, false, true, http.StatusOK},
		{"enable blocked", true, false, true, http.StatusForbidden},
		{"disable blocked", false, true, false, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			slug := "policy_" + uid.DNS1035()
			require.NoError(t, db.Query.UpsertFlag(t.Context(), h.DB.RW(), db.UpsertFlagParams{
				Slug:         slug,
				AllowEnable:  tc.allowEnable,
				AllowDisable: tc.allowDisable,
			}))
			require.NoError(t, db.Query.UpsertWorkspaceFlagOverride(t.Context(), h.DB.RW(), db.UpsertWorkspaceFlagOverrideParams{
				WorkspaceID: workspaceID,
				FlagSlug:    slug,
				Value:       !tc.value,
			}))
			before, err := db.Query.ListClickhouseOutboxByWorkspace(t.Context(), h.DB.RW(), workspaceID)
			require.NoError(t, err)
			for range 2 {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, flagHeaders(key), handler.Request{Slug: slug, Value: tc.value})
				require.Equal(t, tc.status, res.Status, res.RawBody)
			}
			expected := tc.value
			if tc.status != http.StatusOK {
				expected = !tc.value
				after, err := db.Query.ListClickhouseOutboxByWorkspace(t.Context(), h.DB.RW(), workspaceID)
				require.NoError(t, err)
				require.Len(t, after, len(before))
			}
			assertOverride(t, h, workspaceID, slug, true, expected)
		})
	}
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, flagHeaders(key), handler.Request{Slug: "missing_" + uid.DNS1035(), Value: true})
	require.Equal(t, http.StatusNotFound, missing.Status, missing.RawBody)
	require.Contains(t, missing.Body.Error.Type, "flag_not_found")
}

func flagHeaders(key string) http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
}

func TestUpdateFlagRollsBackWhenAuditFails(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: failingAuditlogs{}}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	slug := "rollback_" + uid.DNS1035()
	require.NoError(t, db.Query.UpsertFlag(t.Context(), h.DB.RW(), db.UpsertFlagParams{Slug: slug, AllowEnable: true, AllowDisable: true}))
	require.NoError(t, db.Query.UpsertWorkspaceFlagOverride(t.Context(), h.DB.RW(), db.UpsertWorkspaceFlagOverrideParams{WorkspaceID: workspaceID, FlagSlug: slug, Value: true}))
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":flags#write")
	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, flagHeaders(key), handler.Request{Slug: slug, Value: false})
	require.Equal(t, http.StatusInternalServerError, res.Status, res.RawBody)
	assertOverride(t, h, workspaceID, slug, true, true)
}

type failingAuditlogs struct{}

func (failingAuditlogs) Insert(context.Context, db.DBTX, []auditlog.AuditLog) error {
	return errors.New("audit unavailable")
}

func assertOverride(t *testing.T, h *testutil.Harness, workspaceID, slug string, valid, value bool) {
	t.Helper()
	flags, err := db.Query.ListFlags(t.Context(), h.DB.RW(), workspaceID)
	require.NoError(t, err)
	for _, flag := range flags {
		if flag.Slug == slug {
			require.Equal(t, valid, flag.OverrideValue.Valid)
			require.Equal(t, value, flag.OverrideValue.Bool)
			return
		}
	}
	t.Fatal("flag missing")
}
