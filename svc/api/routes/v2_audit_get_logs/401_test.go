package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test401_NoAuthHeader(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := newRoute(h)
	h.Register(route)

	headers := http.Header{"Content-Type": []string{"application/json"}}
	res := testutil.CallRoute[Request, Response](h, route, headers, Request{})
	// Missing auth header is reported as a 400 by the auth middleware.
	require.Equal(t, http.StatusBadRequest, res.Status)
}

func Test401_InvalidRootKey(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := newRoute(h)
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, bearer("invalid_key_123"), Request{})
	require.Equal(t, http.StatusUnauthorized, res.Status)
}
