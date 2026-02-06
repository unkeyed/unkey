package handler

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

func Test404_KeySpaceNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	_ = h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
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
