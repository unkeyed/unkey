package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

func TestWithReservedHeaderStrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		header   string
		stripped bool
	}{
		{name: "principal stripped", header: "X-Unkey-Principal", stripped: true},
		{name: "request ID stripped", header: "X-Unkey-Request-Id", stripped: true},
		{name: "old hop header stripped", header: "X-Unkey-Frontline-Hops", stripped: true},
		{name: "old force header stripped", header: "X-Unkey-Force-Region", stripped: true},
		{name: "region stripped", header: proxy.HeaderRegion, stripped: true},
		{name: "lowercase canonicalized then stripped", header: "x-unkey-principal", stripped: true},
		{name: "peer metadata kept for handler", header: proxy.HeaderFrontlineMeta, stripped: false},
		{name: "unrelated header kept", header: "Authorization", stripped: false},
		{name: "X-Forwarded-For kept", header: "X-Forwarded-For", stripped: false},
		{name: "user-defined X-Unkey-Foo stripped", header: "X-Unkey-Foo", stripped: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(test.header, "spoofed")
			sess := newMiddlewareSession(t, req)

			var seen http.Header
			next := func(_ context.Context, s *zen.Session) error {
				seen = s.Request().Header.Clone()
				return nil
			}

			err := WithReservedHeaderStrip()(next)(context.Background(), sess)
			require.NoError(t, err)
			canonical := http.CanonicalHeaderKey(test.header)
			if test.stripped {
				require.Empty(t, seen.Get(canonical))
			} else {
				require.Equal(t, "spoofed", seen.Get(canonical))
			}
		})
	}
}

func TestWithReservedHeaderStrip_RemovesReservedTrailers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		trailer  string
		stripped bool
	}{
		{name: "principal stripped", trailer: "X-Unkey-Principal", stripped: true},
		{name: "metadata stripped", trailer: proxy.HeaderFrontlineMeta, stripped: true},
		{name: "region stripped", trailer: proxy.HeaderRegion, stripped: true},
		{name: "lowercase canonicalized then stripped", trailer: "x-unkey-custom", stripped: true},
		{name: "unrelated trailer kept", trailer: "X-Customer-Trailer", stripped: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Trailer = make(http.Header)
			req.Trailer.Set(test.trailer, "spoofed")
			sess := newMiddlewareSession(t, req)

			var seen http.Header
			next := func(_ context.Context, s *zen.Session) error {
				seen = s.Request().Trailer.Clone()
				return nil
			}

			err := WithReservedHeaderStrip()(next)(context.Background(), sess)
			require.NoError(t, err)
			canonical := http.CanonicalHeaderKey(test.trailer)
			if test.stripped {
				require.Empty(t, seen.Get(canonical))
			} else {
				require.Equal(t, "spoofed", seen.Get(canonical))
			}
		})
	}
}

func newMiddlewareSession(t *testing.T, req *http.Request) *zen.Session {
	t.Helper()
	w := httptest.NewRecorder()
	//nolint:exhaustruct
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))
	return sess
}
