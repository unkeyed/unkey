package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test401_InvalidRootKey(t *testing.T) {
	h, route, _ := newRoute(t, true)
	query := Request{Query: "SELECT * FROM ratelimits_v1 WHERE namespace_id = 'rlns_missing'"}

	res := testutil.CallRoute[Request, Response](h, route, auth("invalid_root_key"), query)
	require.Equal(t, 401, res.Status)
}
