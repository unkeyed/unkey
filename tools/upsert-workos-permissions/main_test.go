package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/workos"
)

// TestRunUpsertsPermissions guarantees the tool creates missing WorkOS
// permissions and updates existing ones from the provided definitions.
func TestRunUpsertsPermissions(t *testing.T) {
	type recordedRequest struct {
		Method string
		Path   string
		Query  string
		Body   permissionBody
	}
	type permissionResponse struct {
		Permission permissionBody `json:"permission"`
	}
	type errorResponse struct {
		Message string `json:"message"`
	}
	writeJSON := func(w http.ResponseWriter, status int, body any) {
		w.WriteHeader(status)
		require.NoError(t, json.NewEncoder(w).Encode(body))
	}

	requests := []recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test_key", r.Header.Get("Authorization"))

		recorded := recordedRequest{
			Method: r.Method,
			Path:   r.URL.EscapedPath(),
			Query:  r.URL.RawQuery,
		}
		if r.Body != nil && r.ContentLength != 0 {
			err := json.NewDecoder(r.Body).Decode(&recorded.Body)
			require.NoError(t, err)
		}
		requests = append(requests, recorded)

		switch {
		case r.Method == http.MethodGet && r.URL.EscapedPath() == "/authorization/permissions" && r.URL.Query().Get("after") == "":
			after := "cursor_1"
			response := permissionListResponse{Data: []permissionBody{{Slug: "projects:read"}}}
			response.ListMetadata.After = &after
			writeJSON(w, http.StatusOK, response)
		case r.Method == http.MethodGet && r.URL.EscapedPath() == "/authorization/permissions" && r.URL.Query().Get("after") == "cursor_1":
			writeJSON(w, http.StatusOK, permissionListResponse{Data: []permissionBody{{Slug: "legacy:stale"}}})
		case r.Method == http.MethodGet && r.URL.EscapedPath() == "/authorization/permissions/projects:read":
			writeJSON(w, http.StatusOK, permissionResponse{Permission: permissionBody{Slug: "projects:read"}})
		case r.Method == http.MethodPatch && r.URL.EscapedPath() == "/authorization/permissions/projects:read":
			writeJSON(w, http.StatusOK, permissionResponse{Permission: permissionBody{Slug: "projects:read"}})
		case r.Method == http.MethodGet && r.URL.EscapedPath() == "/authorization/permissions/projects:write":
			writeJSON(w, http.StatusNotFound, errorResponse{Message: "not found"})
		case r.Method == http.MethodPost && r.URL.EscapedPath() == "/authorization/permissions":
			writeJSON(w, http.StatusCreated, permissionResponse{Permission: permissionBody{Slug: "projects:write"}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.EscapedPath())
		}
	}))
	t.Cleanup(server.Close)

	oldBaseURL := workOSAPIBaseURL
	oldClient := workOSHTTPClient
	workOSAPIBaseURL = server.URL
	workOSHTTPClient = server.Client()
	t.Cleanup(func() {
		workOSAPIBaseURL = oldBaseURL
		workOSHTTPClient = oldClient
	})

	definitions := []workos.PermissionDefinition{
		{
			Slug:        "projects:read",
			Name:        "Read projects",
			Description: "Read project metadata.",
		},
		{
			Slug:        "projects:write",
			Name:        "Write projects",
			Description: "Create and update project records.",
		},
	}

	var out bytes.Buffer
	err := run(t.Context(), config{apiKey: "test_key", dryRun: false}, definitions, &out)
	require.NoError(t, err)
	require.Equal(t, "updated projects:read\ncreated projects:write\nunmanaged legacy:stale\n", out.String())

	require.Len(t, requests, 6)
	require.Equal(t, http.MethodGet, requests[0].Method)
	require.Equal(t, "/authorization/permissions", requests[0].Path)
	require.Empty(t, requests[0].Query)
	require.Equal(t, http.MethodGet, requests[1].Method)
	require.Equal(t, "/authorization/permissions", requests[1].Path)
	require.Equal(t, "after=cursor_1", requests[1].Query)
	require.Equal(t, http.MethodGet, requests[2].Method)
	require.Equal(t, http.MethodPatch, requests[3].Method)
	require.Equal(t, permissionBody{
		Name:        "Read projects",
		Description: "Read project metadata.",
	}, requests[3].Body)
	require.Equal(t, http.MethodGet, requests[4].Method)
	require.Equal(t, http.MethodPost, requests[5].Method)
	require.Equal(t, permissionBody{
		Slug:        "projects:write",
		Name:        "Write projects",
		Description: "Create and update project records.",
	}, requests[5].Body)
}

// TestRunDryRunDoesNotRequireAPIKey guarantees local previews cannot
// accidentally call WorkOS or require credentials.
func TestRunDryRunDoesNotRequireAPIKey(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := run(t.Context(), config{apiKey: "", dryRun: true}, []workos.PermissionDefinition{
		{
			Slug:        "admin:*",
			Name:        "Admin",
			Description: "Grants full administrative access.",
		},
	}, &out)
	require.NoError(t, err)
	require.Equal(t, "would upsert admin:* (Admin)\n", out.String())
}

// TestRunRequiresAPIKey guarantees non-dry-run execution fails before making
// WorkOS requests when credentials are absent.
func TestRunRequiresAPIKey(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := run(t.Context(), config{apiKey: "", dryRun: false}, []workos.PermissionDefinition{
		{
			Slug:        "admin:*",
			Name:        "Admin",
			Description: "Grants full administrative access.",
		},
	}, &out)
	require.ErrorContains(t, err, "api key is required")
	require.Empty(t, out.String())
}
