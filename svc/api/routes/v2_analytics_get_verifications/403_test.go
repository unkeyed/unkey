package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test403_NoAnalyticsPermission(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

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

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications_v1",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 403, res.Status)
}
