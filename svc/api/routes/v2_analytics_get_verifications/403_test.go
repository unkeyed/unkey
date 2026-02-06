package handler

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

func Test403_NoAnalyticsPermission(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	_ = h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	// Create root key WITHOUT read_analytics permission
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

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

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications_v1",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 403, res.Status)
}

func Test403_WrongApiPermission(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api1 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	api2 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	// Create root key with permission only for api1
	rootKey := h.CreateRootKey(workspace.ID, "api."+api1.ID+".read_analytics")

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

	// Query filtering by api2's key_space_id but user only has permission for api1
	req := Request{
		Query: fmt.Sprintf("SELECT COUNT(*) FROM key_verifications_v1 WHERE key_space_id = '%s'", api2.KeyAuthID.String),
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 403, res.Status)
}
