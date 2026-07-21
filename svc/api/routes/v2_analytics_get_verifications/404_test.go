package handler

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func Test404_KeySpaceNotFound(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api."+api.ID+".read_analytics")

	route := &Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Query with non-existent key_space_id
	req := Request{
		Query: fmt.Sprintf("SELECT COUNT(*) FROM key_verifications_v1 WHERE key_space_id = '%s'", "ks_nonexistent123"),
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 404, res.Status) // Key space not found
}

func Test404_MixedKeySpacesFailClosed(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	owned := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	unauthorized := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	foreignWorkspace := h.CreateWorkspace()
	foreign := h.CreateApi(seed.CreateApiRequest{WorkspaceID: foreignWorkspace.ID})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api."+owned.ID+".read_analytics")
	route := &Handler{DB: h.DB, AnalyticsConnectionManager: h.AnalyticsConnectionManager, Caches: h.Caches}
	h.Register(route)
	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Security guarantee: one foreign, missing, or unauthorized ID makes the complete request fail closed.
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM key_verifications_v1 AS v WHERE v.key_space_id IN ('%s', '%s', '%s', '%s')",
		owned.KeyAuthID.String,
		unauthorized.KeyAuthID.String,
		foreign.KeyAuthID.String,
		"ks_missing",
	)
	res := testutil.CallRoute[Request, openapi.NotFoundErrorResponse](h, route, headers, Request{Query: query})
	require.Equal(t, http.StatusNotFound, res.Status)

	// Security guarantee: a foreign key space is externally indistinguishable from a missing one.
	foreignRes := testutil.CallRoute[Request, openapi.NotFoundErrorResponse](h, route, headers, Request{Query: fmt.Sprintf(
		"SELECT COUNT(*) FROM key_verifications_v1 WHERE key_space_id = '%s'", foreign.KeyAuthID.String,
	)})
	missingRes := testutil.CallRoute[Request, openapi.NotFoundErrorResponse](h, route, headers, Request{Query: "SELECT COUNT(*) FROM key_verifications_v1 WHERE key_space_id = 'ks_also_missing'"})
	require.Equal(t, http.StatusNotFound, foreignRes.Status)
	require.Equal(t, missingRes.Status, foreignRes.Status)
	require.Equal(t, missingRes.Body.Error.Type, foreignRes.Body.Error.Type)
}
