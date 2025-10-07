package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func Test401_NoAuthHeader(t *testing.T) {
	h := testutil.NewHarness(t)

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
		"Content-Type": []string{"application/json"},
	}

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 401, res.Status)
}

func Test401_InvalidRootKey(t *testing.T) {
	h := testutil.NewHarness(t)

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
		"Authorization": []string{"Bearer invalid_key_123"},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 401, res.Status)
}
