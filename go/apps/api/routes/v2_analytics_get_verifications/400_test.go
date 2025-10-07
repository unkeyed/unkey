package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func Test400_InvalidRequestBody(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

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

	// Empty query should fail validation
	req := Request{
		Query: "",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 400, res.Status)
}

func Test400_MalformedJSON(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

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

	// Malformed JSON
	req := Request{
		Query: "SELECT * FROM key_verifications WHERE invalid syntax",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 400, res.Status)
}
