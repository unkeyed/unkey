package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test401_InvalidRootKey(t *testing.T) {
	h, route, _ := newRoute(t, true)

	res := testutil.CallRoute[Request, Response](h, route, auth("unkey_not_a_real_key"), Request{
		Query: "SELECT count() FROM runtime_logs_v1",
	})
	require.Equal(t, 401, res.Status)
}

func Test401_MissingAuthHeader(t *testing.T) {
	h, route, _ := newRoute(t, true)

	res := testutil.CallRoute[Request, Response](h, route, http.Header{"Content-Type": {"application/json"}}, Request{
		Query: "SELECT count() FROM runtime_logs_v1",
	})
	require.Equal(t, 400, res.Status)
}
