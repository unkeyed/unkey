package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func TestGetFlagsUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(t, h, h.DB)
	h.Register(route)

	res := callGetFlags(h, route, bearer("invalid_token"))
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
}
