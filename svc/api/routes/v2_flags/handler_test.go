package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_flags"
)

func TestListPreservesFalseOverrideAndWorkspaceIsolation(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	flagID := uid.New(uid.TestPrefix)
	slug := "test-" + flagID
	_, err := h.DB.RW().ExecContext(t.Context(), "INSERT INTO flags (id, slug, description, default_value) VALUES (?, ?, 'test', true)", flagID, slug)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := h.DB.RW().ExecContext(context.Background(), "DELETE FROM workspace_flag_overrides WHERE flag_id = ?", flagID)
		require.NoError(t, err)
		_, err = h.DB.RW().ExecContext(context.Background(), "DELETE FROM flags WHERE id = ?", flagID)
		require.NoError(t, err)
	})
	_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO workspace_flag_overrides (workspace_id, flag_id, value) VALUES (?, ?, false)", workspaceID, flagID)
	require.NoError(t, err)
	p := &principal.Principal{Type: principal.TypeJWT, Subject: principal.Subject{ID: "test-user", Type: principal.SubjectTypeUser}, Source: principal.JWTSource{Roles: []string{"admin"}}, AuthorizedWorkspaceID: workspaceID}
	route := &handler.ListHandler{DB: h.DB}
	h.Register(route, append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error { s.SetPrincipal(p); return next(ctx, s) }
	})...)
	type response struct {
		Data []struct {
			Slug        string          `json:"slug"`
			Value       json.RawMessage `json:"value"`
			HasOverride bool            `json:"hasOverride"`
		} `json:"data"`
	}
	for _, own := range []bool{true, false} {
		if !own {
			p.AuthorizedWorkspaceID = uid.New(uid.WorkspacePrefix)
		}
		res := testutil.CallRoute[struct{}, response](h, route, http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer test"}}, struct{}{})
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		found := false
		for _, flag := range res.Body.Data {
			if flag.Slug == slug {
				found = true
				require.Equal(t, own, flag.HasOverride)
				if own {
					require.JSONEq(t, "false", string(flag.Value))
				} else {
					require.JSONEq(t, "true", string(flag.Value))
				}
			}
		}
		require.True(t, found)
	}
}

func TestSetOverrideRejectsNumbers(t *testing.T) {
	h, p, slug := mutationFixture(t)
	route := &handler.SetHandler{DB: h.DB}
	registerPrincipal(h, route, p)
	res := testutil.CallRoute[map[string]any, overrideResponse](h, route, authHeaders(), map[string]any{"slug": slug, "value": 0})
	require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
}

func TestRemoveOverrideRestoresDefaultAndIsIdempotent(t *testing.T) {
	h, p, slug := mutationFixture(t)
	set := &handler.SetHandler{DB: h.DB}
	remove := &handler.RemoveHandler{DB: h.DB}
	registerPrincipal(h, set, p)
	registerPrincipal(h, remove, p)
	res := testutil.CallRoute[map[string]any, overrideResponse](h, set, authHeaders(), map[string]any{"slug": slug, "value": false})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.True(t, res.Body.Data.HasOverride)
	require.JSONEq(t, "false", string(res.Body.Data.Value))
	for range 2 {
		res := testutil.CallRoute[map[string]string, overrideResponse](h, remove, authHeaders(), map[string]string{"slug": slug})
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		require.False(t, res.Body.Data.HasOverride)
		require.JSONEq(t, "true", string(res.Body.Data.Value))
	}
}

func TestSetRejectsInvalidValuesWithoutChangingOverride(t *testing.T) {
	h, p, slug := mutationFixture(t)
	set := &handler.SetHandler{DB: h.DB}
	list := &handler.ListHandler{DB: h.DB}
	registerPrincipal(h, set, p)
	registerPrincipal(h, list, p)
	for _, body := range []string{
		`{"slug":"` + slug + `","value":null}`,
		`{"slug":"` + slug + `","value":"false"}`,
		`{"slug":"` + slug + `","value":1}`,
		`{"slug":"` + slug + `","value":{}}`,
		`{"slug":"` + slug + `","value":[]}`,
		`{"slug":"` + slug + `"}`,
		`{"slug":"` + slug + `","value":true,"workspaceId":"other"}`,
	} {
		t.Run(body, func(t *testing.T) {
			res := testutil.CallRoute[json.RawMessage, overrideResponse](h, set, authHeaders(), json.RawMessage(body))
			require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
		})
	}
	res := testutil.CallRoute[struct{}, listResponse](h, list, authHeaders(), struct{}{})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	for _, flag := range res.Body.Data {
		if flag.Slug == slug {
			require.False(t, flag.HasOverride)
			require.JSONEq(t, "true", string(flag.Value))
			return
		}
	}
	t.Fatal("flag missing from list")
}

func TestEnrollmentPermissionsAreIndependent(t *testing.T) {
	h, p, slug := mutationFixture(t)
	set := &handler.SetHandler{DB: h.DB}
	remove := &handler.RemoveHandler{DB: h.DB}
	registerPrincipal(h, set, p)
	registerPrincipal(h, remove, p)
	for _, tc := range []struct {
		name          string
		optIn, optOut bool
	}{
		{"neither", false, false}, {"opt-in only", true, false},
		{"opt-out only", false, true}, {"both", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE flags SET allow_opt_in = ?, allow_opt_out = ? WHERE slug = ?", tc.optIn, tc.optOut, slug)
			require.NoError(t, err)
			setResult := testutil.CallRoute[map[string]any, overrideResponse](h, set, authHeaders(), map[string]any{"slug": slug, "value": false})
			want := http.StatusForbidden
			if tc.optIn {
				want = http.StatusOK
			}
			require.Equal(t, want, setResult.Status, "%s", setResult.RawBody)
			removeResult := testutil.CallRoute[map[string]string, overrideResponse](h, remove, authHeaders(), map[string]string{"slug": slug})
			want = http.StatusForbidden
			if tc.optOut {
				want = http.StatusOK
			}
			require.Equal(t, want, removeResult.Status, "%s", removeResult.RawBody)
		})
	}
}

func TestOnlyWorkspaceAdminsCanMutateFlags(t *testing.T) {
	h, p, slug := mutationFixture(t)
	set := &handler.SetHandler{DB: h.DB}
	remove := &handler.RemoveHandler{DB: h.DB}
	list := &handler.ListHandler{DB: h.DB}
	registerPrincipal(h, set, p)
	registerPrincipal(h, remove, p)
	registerPrincipal(h, list, p)
	p.Source = principal.JWTSource{Roles: []string{"member"}}
	res := testutil.CallRoute[struct{}, listResponse](h, list, authHeaders(), struct{}{})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	for _, route := range []zen.Route{set, remove} {
		body := map[string]any{"slug": slug}
		if route == set {
			body["value"] = true
		}
		res := testutil.CallRoute[map[string]any, overrideResponse](h, route, authHeaders(), body)
		require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	}
	p.Type = principal.TypeAPIKey
	p.Source = principal.KeySource{}
	for _, route := range []zen.Route{list, set, remove} {
		body := map[string]any{}
		if route != list {
			body["slug"] = slug
		}
		if route == set {
			body["value"] = true
		}
		res := testutil.CallRoute[map[string]any, json.RawMessage](h, route, authHeaders(), body)
		require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	}
}

func TestFlagsRequireAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	for _, route := range []zen.Route{&handler.ListHandler{DB: h.DB}, &handler.SetHandler{DB: h.DB}, &handler.RemoveHandler{DB: h.DB}} {
		h.Register(route)
		body := map[string]any{}
		if route.Path() != "/v2/flags.listFlags" {
			body["slug"] = "example"
		}
		if route.Path() == "/v2/flags.setOverride" {
			body["value"] = true
		}
		res := testutil.CallRoute[map[string]any, json.RawMessage](h, route, authHeaders(), body)
		require.Equal(t, http.StatusUnauthorized, res.Status, "%s", res.RawBody)
	}
}

func TestTrueOverridePersistsAndRemovalIsWorkspaceScoped(t *testing.T) {
	h, p, slug := mutationFixture(t)
	_, err := h.DB.RW().ExecContext(t.Context(), `UPDATE flags SET default_value = false WHERE slug = ?`, slug)
	require.NoError(t, err)
	set := &handler.SetHandler{DB: h.DB}
	remove := &handler.RemoveHandler{DB: h.DB}
	list := &handler.ListHandler{DB: h.DB}
	registerPrincipal(h, set, p)
	registerPrincipal(h, remove, p)
	registerPrincipal(h, list, p)
	ownWorkspace := p.AuthorizedWorkspaceID
	otherWorkspace := uid.New(uid.WorkspacePrefix)
	for _, workspace := range []string{ownWorkspace, otherWorkspace} {
		p.AuthorizedWorkspaceID = workspace
		for range 2 {
			res := testutil.CallRoute[map[string]any, overrideResponse](h, set, authHeaders(), map[string]any{"slug": slug, "value": true})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		}
	}
	removed := testutil.CallRoute[map[string]string, overrideResponse](h, remove, authHeaders(), map[string]string{"slug": slug})
	require.Equal(t, http.StatusOK, removed.Status, "%s", removed.RawBody)
	for _, workspace := range []string{ownWorkspace, otherWorkspace} {
		p.AuthorizedWorkspaceID = workspace
		res := testutil.CallRoute[struct{}, listResponse](h, list, authHeaders(), struct{}{})
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		found := false
		for _, flag := range res.Body.Data {
			if flag.Slug != slug {
				continue
			}
			found = true
			require.Equal(t, workspace == ownWorkspace, flag.HasOverride)
			if workspace == ownWorkspace {
				require.JSONEq(t, "true", string(flag.Value))
			} else {
				require.JSONEq(t, "false", string(flag.Value))
			}
		}
		require.True(t, found)
	}
}

type listResponse struct {
	Data []struct {
		Slug        string          `json:"slug"`
		Value       json.RawMessage `json:"value"`
		HasOverride bool            `json:"hasOverride"`
	} `json:"data"`
}

type overrideResponse struct {
	Data struct {
		Value       json.RawMessage `json:"value"`
		HasOverride bool            `json:"hasOverride"`
	} `json:"data"`
}

func mutationFixture(t *testing.T) (*testutil.Harness, *principal.Principal, string) {
	t.Helper()
	h := testutil.NewHarness(t)
	id := uid.New(uid.TestPrefix)
	slug := strings.ToLower(strings.ReplaceAll(id, "_", "-"))
	_, err := h.DB.RW().ExecContext(t.Context(), "INSERT INTO flags (id, slug, description, default_value, allow_opt_in, allow_opt_out) VALUES (?, ?, 'test', true, true, true)", id, slug)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := h.DB.RW().ExecContext(context.Background(), "DELETE FROM workspace_flag_overrides WHERE flag_id = ?", id)
		require.NoError(t, err)
		_, err = h.DB.RW().ExecContext(context.Background(), "DELETE FROM flags WHERE id = ?", id)
		require.NoError(t, err)
	})
	p := &principal.Principal{Type: principal.TypeJWT, Subject: principal.Subject{ID: "test-user", Type: principal.SubjectTypeUser}, Source: principal.JWTSource{Roles: []string{"admin"}}, AuthorizedWorkspaceID: h.Resources().UserWorkspace.ID}
	return h, p, slug
}

func registerPrincipal(h *testutil.Harness, route zen.Route, p *principal.Principal) {
	h.Register(route, append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error { s.SetPrincipal(p); return next(ctx, s) }
	})...)
}

func authHeaders() http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer test"}}
}
