package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionRejectionWritesNothing(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		Clock:         h.Clock,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	projectID, portalID, keyspaceID := sessionURNFixture(t, h, "no-write")
	grants := append([]string{sessionWriteGrant(workspaceID, projectID, portalID)}, canonicalReadGrants(workspaceID, projectID, keyspaceID)...)
	rootKey := h.CreateRootKey(workspaceID, grants...)
	externalID := uid.New(uid.TestPrefix)
	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}, handler.Request{
		Portal:     "no-write",
		ExternalId: externalID,
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead, openapi.KeysReroll},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.NotEmpty(t, res.Body.Error.Detail)
	require.Zero(t, countPortalSessions(t, h, workspaceID, externalID))
	require.Zero(t, countAuditEntriesMentioning(t, h, workspaceID, externalID))
	requireNoSessionCredential(t, res.RawBody)
	require.Empty(t, res.Headers.Get("Set-Cookie"))
}

func TestCreateSessionRejectsUnavailablePortal(t *testing.T) {
	for _, orphan := range []bool{false, true} {
		name := "disabled"
		if orphan {
			name = "orphan keyspace"
		}
		t.Run(name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := &handler.Handler{
				DB:            h.DB,
				Auditlogs:     h.Auditlogs,
				Clock:         h.Clock,
				PortalBaseURL: "https://portal.unkey.com",
			}
			h.Register(route)
			workspaceID := h.Resources().UserWorkspace.ID
			api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
			keyspaceID := api.KeyAuthID.String
			if orphan {
				keyspaceID = uid.New(uid.KeySpacePrefix)
				require.NoError(t, db.Query.InsertKeySpace(context.Background(), h.DB.RW(), db.InsertKeySpaceParams{
					ID:          keyspaceID,
					WorkspaceID: workspaceID,
					ProjectID:   api.ProjectID,
					CreatedAtM:  h.Clock.Now().UnixMilli(),
				}))
			}
			stored := h.CreatePortal(seed.CreatePortalRequest{
				ID:          uid.New(uid.PortalPrefix),
				WorkspaceID: workspaceID,
				ProjectID:   api.ProjectID,
				Slug:        "unavailable",
				DisplayName: "Unavailable",
				Enabled:     orphan,
				KeyAuthID:   sql.NullString{String: keyspaceID, Valid: true},
			})
			grants := append([]string{sessionWriteGrant(workspaceID, api.ProjectID, stored.ID)}, canonicalReadGrants(workspaceID, api.ProjectID, keyspaceID)...)
			rootKey := h.CreateRootKey(workspaceID, grants...)
			externalID := uid.New(uid.TestPrefix)
			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer " + rootKey},
			}, handler.Request{
				Portal:     stored.Slug,
				ExternalId: externalID,
				Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
			})
			require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
			want := "Portal is disabled."
			if orphan {
				want = "Portal is not available: the API it uses no longer exists."
			}
			require.Equal(t, want, res.Body.Error.Detail)
			require.Zero(t, countPortalSessions(t, h, workspaceID, externalID))
			require.Zero(t, countAuditEntriesMentioning(t, h, workspaceID, externalID))
			requireNoSessionCredential(t, res.RawBody)
		})
	}
}

func TestCreateSessionCrossProjectKeyspaceAfterRedeploy(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		Clock:         h.Clock,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	inside := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "Other",
		Slug:        "other-" + uid.DNS1035(),
	})
	outside := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspaceID,
		ProjectID:   otherProject.ID,
	})
	app := seedAppWithKeyspaces(t, h, workspaceID, "redeploy-authz", inside.ProjectID, []string{inside.KeyAuthID.String})
	stored := h.SeedPortal(t, workspaceID, "redeploy-authz", "Redeploy", appMapping(app.AppID), nil, nil)
	grants := append([]string{sessionWriteGrant(workspaceID, inside.ProjectID, stored.ID)}, canonicalReadGrants(workspaceID, inside.ProjectID, inside.KeyAuthID.String)...)
	grants = append(grants, canonicalReadGrants(workspaceID, outside.ProjectID, outside.KeyAuthID.String)...)
	rootKey := h.CreateRootKey(workspaceID, grants...)
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + rootKey}}
	req := handler.Request{
		Portal:     stored.Slug,
		ExternalId: uid.New(uid.TestPrefix),
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
	}
	before := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, before.Status, "%s", before.RawBody)
	for _, keyspaces := range [][]string{{outside.KeyAuthID.String}, {inside.KeyAuthID.String, outside.KeyAuthID.String}} {
		redeployAppWithKeyspaces(t, h, workspaceID, app, keyspaces)
		logs := loggertest.Install(t)
		req.ExternalId = uid.New(uid.TestPrefix)
		after := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, after.Status, "%s", after.RawBody)
		require.Equal(t, "Portal is not available: it serves a keyspace outside its project. Fix the app's deployment configuration.", after.Body.Error.Detail)
		for _, secret := range []string{outside.KeyAuthID.String, outside.ProjectID, inside.ProjectID} {
			require.NotContains(t, after.RawBody, secret)
		}
		require.Zero(t, countPortalSessions(t, h, workspaceID, req.ExternalId))
		require.Zero(t, countAuditEntriesMentioning(t, h, workspaceID, req.ExternalId))
		requireNoSessionCredential(t, after.RawBody)
		record := logs.Find(t, "portal resolved a keyspace outside its project")
		attrs := loggertest.FlatAttrs(record)
		require.Equal(t, stored.ID, attrs["portal_id"])
		require.Equal(t, inside.ProjectID, attrs["portal_project_id"])
		require.Equal(t, outside.KeyAuthID.String, attrs["keyspace_id"])
		require.Equal(t, outside.ProjectID, attrs["keyspace_project_id"])
	}
}

func requireNoSessionCredential(t *testing.T, rawBody string) {
	t.Helper()
	var response handler.Response
	require.NoError(t, json.Unmarshal([]byte(rawBody), &response))
	require.Empty(t, response.Data.Id)
	require.Empty(t, response.Data.Url)
}
