package handler

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// TestCanonicalLogReadScopesRowsByOwnership guarantees that exact, wildcard,
// and union grants expose only rows covered by ownership-derived log URNs.
func TestCanonicalLogReadScopesRowsByOwnership(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	allowedAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	forbiddenProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Forbidden project",
		Slug:        uid.New("project"),
	})
	forbiddenAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID, ProjectID: forbiddenProject.ID})
	foreignWorkspace := h.CreateWorkspace()
	foreignAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: foreignWorkspace.ID})
	h.SetupAnalytics(workspace.ID)

	for _, verification := range []schema.KeyVerification{
		verificationRow(workspace.ID, allowedAPI.KeyAuthID.String),
		verificationRow(workspace.ID, forbiddenAPI.KeyAuthID.String),
		verificationRow(foreignWorkspace.ID, foreignAPI.KeyAuthID.String),
	} {
		h.KeyVerifications.Buffer(verification)
	}

	route := &Handler{DB: h.DB, AnalyticsConnectionManager: h.AnalyticsConnectionManager, Caches: h.Caches}
	h.Register(route)
	query := Request{Query: "SELECT key_space_id FROM key_verifications_v1"}
	wildcardHeaders := analyticsHeaders(h.CreateRootKey(workspace.ID, "api.*.read_analytics"))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, wildcardHeaders, query)
		require.Equal(c, http.StatusOK, res.Status)
		require.Len(c, res.Body.Data, 2)
	}, 30*time.Second, time.Second)

	permission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/logs#read", workspace.ID, allowedAPI.ProjectID, allowedAPI.KeyAuthID.String)
	res := testutil.CallRoute[Request, Response](h, route, analyticsHeaders(h.CreateRootKey(workspace.ID, permission)), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"key_space_id": allowedAPI.KeyAuthID.String}}, res.Body.Data)

	logDescendants := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/logs/**#read", workspace.ID, allowedAPI.ProjectID, allowedAPI.KeyAuthID.String)
	res = testutil.CallRoute[Request, Response](h, route, analyticsHeaders(h.CreateRootKey(workspace.ID, logDescendants)), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"key_space_id": allowedAPI.KeyAuthID.String}}, res.Body.Data)

	projectPermission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/*/logs#read", workspace.ID, allowedAPI.ProjectID)
	res = testutil.CallRoute[Request, Response](h, route, analyticsHeaders(h.CreateRootKey(workspace.ID, projectPermission)), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"key_space_id": allowedAPI.KeyAuthID.String}}, res.Body.Data)

	wrongOwnerPermission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/logs#read", workspace.ID, forbiddenProject.ID, allowedAPI.KeyAuthID.String)
	res = testutil.CallRoute[Request, Response](h, route, analyticsHeaders(h.CreateRootKey(workspace.ID, wrongOwnerPermission)), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.Empty(t, res.Body.Data)

	missingPermission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/logs#read", workspace.ID, allowedAPI.ProjectID, uid.New(uid.KeySpacePrefix))
	res = testutil.CallRoute[Request, Response](h, route, analyticsHeaders(h.CreateRootKey(workspace.ID, missingPermission)), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.Empty(t, res.Body.Data)

	unionRootKey := h.CreateRootKey(workspace.ID, permission, "api."+forbiddenAPI.ID+".read_analytics")
	res = testutil.CallRoute[Request, Response](h, route, analyticsHeaders(unionRootKey), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.ElementsMatch(t, []map[string]any{
		{"key_space_id": allowedAPI.KeyAuthID.String},
		{"key_space_id": forbiddenAPI.KeyAuthID.String},
	}, res.Body.Data)

	_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE key_auth SET deleted_at_m = ? WHERE id = ?", time.Now().UnixMilli(), allowedAPI.KeyAuthID.String)
	require.NoError(t, err)
	res = testutil.CallRoute[Request, Response](h, route, analyticsHeaders(h.CreateRootKey(workspace.ID, permission)), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"key_space_id": allowedAPI.KeyAuthID.String}}, res.Body.Data)

	workspacePermission := fmt.Sprintf("unkey:v1:%s:projects/*/keyspaces/*/logs#read", workspace.ID)
	res = testutil.CallRoute[Request, Response](h, route, analyticsHeaders(h.CreateRootKey(workspace.ID, workspacePermission)), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.ElementsMatch(t, []map[string]any{
		{"key_space_id": allowedAPI.KeyAuthID.String},
		{"key_space_id": forbiddenAPI.KeyAuthID.String},
	}, res.Body.Data)

	globalPermission := fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID)
	res = testutil.CallRoute[Request, Response](h, route, analyticsHeaders(h.CreateRootKey(workspace.ID, globalPermission)), query)
	require.Equal(t, http.StatusOK, res.Status, "body: %s", res.RawBody)
	require.ElementsMatch(t, []map[string]any{
		{"key_space_id": allowedAPI.KeyAuthID.String},
		{"key_space_id": forbiddenAPI.KeyAuthID.String},
	}, res.Body.Data)
}

// verificationRow creates one analytics row for an authorization boundary test.
func verificationRow(workspaceID, keyspaceID string) schema.KeyVerification {
	return schema.KeyVerification{
		RequestID:   uid.New(uid.RequestPrefix),
		Time:        time.Now().UnixMilli(),
		WorkspaceID: workspaceID,
		KeySpaceID:  keyspaceID,
		KeyID:       uid.New(uid.KeyPrefix),
		Region:      "us-east-1",
		Outcome:     "VALID",
	}
}

// analyticsHeaders authenticates an analytics route call with a root key.
func analyticsHeaders(rootKey string) http.Header {
	return http.Header{
		"Authorization": {"Bearer " + rootKey},
		"Content-Type":  {"application/json"},
	}
}
