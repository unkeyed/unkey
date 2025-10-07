package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func Test404_AnalyticsNotConfigured(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	_ = h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	// Do NOT set up ClickHouse workspace settings
	// This will cause GetConnection to fail

	route := &Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		ClickHouse: h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 404, res.Status)
}
