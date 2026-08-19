package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test401_InvalidRootKey guarantees an unknown root key cannot reach ClickHouse.
func Test401_InvalidRootKey(t *testing.T) {
	h, route, _ := newRoute(t, true)

	res := testutil.CallRoute[Request, Response](h, route, auth("unkey_not_a_real_key"), Request{
		Query: "SELECT count() FROM gateway_requests_v1",
	})
	require.Equal(t, 401, res.Status)
}

// Test401_MissingAuthHeader guarantees a request without credentials is
// refused. A missing header is a malformed request, so it returns 400 rather
// than 401, which matches the other analytics endpoints.
func Test401_MissingAuthHeader(t *testing.T) {
	h, route, _ := newRoute(t, true)

	res := testutil.CallRoute[Request, Response](h, route, http.Header{"Content-Type": {"application/json"}}, Request{
		Query: "SELECT count() FROM gateway_requests_v1",
	})
	require.Equal(t, 400, res.Status)
}
