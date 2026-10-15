package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	authworkos "github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_update_flag"
)

func TestUpdateFlagAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Flags: h.Flags}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	other := h.CreateWorkspace()
	slug := "auth_" + uid.DNS1035()
	require.NoError(t, db.Query.UpsertFlag(t.Context(), h.DB.RW(), db.UpsertFlagParams{Slug: slug, AllowOptIn: true, AllowOptOut: true}))
	require.NoError(t, db.Query.UpsertWorkspaceFlagOverride(t.Context(), h.DB.RW(), db.UpsertWorkspaceFlagOverrideParams{WorkspaceID: workspaceID, FlagSlug: slug, Enabled: false}))
	for _, tc := range []struct {
		name, key string
		status    int
	}{
		{"no permissions", h.CreateRootKey(workspaceID), http.StatusForbidden},
		{"read only", h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":flags#read"), http.StatusForbidden},
		{"other workspace", h.CreateRootKey(workspaceID, "unkey:v1:"+other.ID+":flags#write"), http.StatusForbidden},
		{"invalid token", "invalid_token", http.StatusUnauthorized},
		{"missing token", "", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			headers := flagHeaders(tc.key)
			if tc.key == "" {
				headers.Del("Authorization")
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Slug: slug, Enabled: true})
			require.Equal(t, tc.status, res.Status, res.RawBody)
			assertOverride(t, h, workspaceID, slug, true, false)
			assertOverride(t, h, other.ID, slug, false, false)
		})
	}
}

func TestUpdateFlagRejectsWorkspaceSelectorAndMissingEnabled(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Flags: h.Flags}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	other := h.CreateWorkspace()
	slug := "input_" + uid.DNS1035()
	require.NoError(t, db.Query.UpsertFlag(t.Context(), h.DB.RW(), db.UpsertFlagParams{Slug: slug, AllowOptIn: true, AllowOptOut: true}))
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":flags#write")
	type workspaceSelector struct {
		Slug        string `json:"slug"`
		Enabled     bool   `json:"enabled"`
		WorkspaceID string `json:"workspaceId"`
	}
	selected := testutil.CallRoute[workspaceSelector, handler.Response](h, route, flagHeaders(key), workspaceSelector{Slug: slug, Enabled: true, WorkspaceID: other.ID})
	require.Equal(t, http.StatusBadRequest, selected.Status, selected.RawBody)
	type missingEnabled struct {
		Slug string `json:"slug"`
	}
	missing := testutil.CallRoute[missingEnabled, handler.Response](h, route, flagHeaders(key), missingEnabled{Slug: slug})
	require.Equal(t, http.StatusBadRequest, missing.Status, missing.RawBody)
	assertOverride(t, h, workspaceID, slug, false, false)
	assertOverride(t, h, other.ID, slug, false, false)
}

func TestUpdateFlagDashboardRoles(t *testing.T) {
	for _, role := range []string{"admin", "developer", "basic_member", "viewer", "unknown"} {
		t.Run(role, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Flags: h.Flags}
			workspaceID := h.Resources().UserWorkspace.ID
			slug := "role_" + uid.DNS1035()
			require.NoError(t, db.Query.UpsertFlag(t.Context(), h.DB.RW(), db.UpsertFlagParams{Slug: slug, AllowOptIn: true, AllowOptOut: true}))
			require.NoError(t, db.Query.UpsertWorkspaceFlagOverride(t.Context(), h.DB.RW(), db.UpsertWorkspaceFlagOverrideParams{WorkspaceID: workspaceID, FlagSlug: slug, Enabled: false}))
			resolver := authworkos.NewRoleMappingResolver(jwtResolver{principal: &authprincipal.Principal{
				Version:               authprincipal.Version,
				Subject:               authprincipal.Subject{ID: uid.Secure(), Name: "dashboard user", Type: authprincipal.SubjectTypeUser},
				Type:                  authprincipal.TypeJWT,
				Source:                authprincipal.JWTSource{Roles: []string{role}},
				AuthorizedWorkspaceID: workspaceID,
			}})
			stack := append([]zen.Middleware{}, h.PublicMiddleware()...)
			stack = append(stack, func(next zen.HandleFunc) zen.HandleFunc {
				return func(ctx context.Context, s *zen.Session) error {
					principal, err := resolver.Resolve(ctx, s)
					if err != nil {
						return err
					}
					s.SetPrincipal(principal)
					return next(ctx, s)
				}
			})
			h.Register(route, stack...)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, flagHeaders("dashboard_jwt"), handler.Request{Slug: slug, Enabled: true})
			status := http.StatusForbidden
			if role != "unknown" && role != "viewer" {
				status = http.StatusOK
			}
			require.Equal(t, status, res.Status, res.RawBody)
			assertOverride(t, h, workspaceID, slug, true, status == http.StatusOK)
		})
	}
}

type jwtResolver struct{ principal *authprincipal.Principal }

func (r jwtResolver) Resolve(context.Context, *zen.Session) (*authprincipal.Principal, error) {
	return r.principal, nil
}
