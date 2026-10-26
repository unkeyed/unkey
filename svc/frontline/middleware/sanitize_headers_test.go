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
		{name: "underscore principal stripped", header: "X_Unkey_Principal", stripped: true},
		{name: "mixed separator principal stripped", header: "X-Unkey_Principal", stripped: true},
		{name: "underscore prefix stripped", header: "x_unkey-Principal", stripped: true},
		{name: "metadata alias stripped", header: "X_Unkey_Frontline_Meta", stripped: true},
		{name: "peer metadata kept for handler", header: proxy.HeaderFrontlineMeta, stripped: false},
		{name: "unrelated header kept", header: "Authorization", stripped: false},
		{name: "unrelated underscore header kept", header: "X_Customer_Principal", stripped: false},
		{name: "underscore prefix extension stripped", header: "x_unkeyabc", stripped: true},
		{name: "hyphen prefix extension stripped", header: "X-Unkeyed", stripped: true},
		{name: "bare hyphen prefix stripped", header: "X-Unkey", stripped: true},
		{name: "bare underscore prefix stripped", header: "x_unkey", stripped: true},
		{name: "unkey hyphen prefix stripped", header: "Unkey-X", stripped: true},
		{name: "unkey underscore prefix stripped", header: "uNkEy_x", stripped: true},
		{name: "unkey without separator stripped", header: "Unkeyed", stripped: true},
		{name: "bare unkey prefix stripped", header: "Unkey", stripped: true},
		{name: "short prefix kept", header: "X-Unke", stripped: false},
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
		{name: "underscore principal stripped", trailer: "X_Unkey_Principal", stripped: true},
		{name: "mixed separator principal stripped", trailer: "X-Unkey_Principal", stripped: true},
		{name: "bare hyphen prefix stripped", trailer: "X-Unkey", stripped: true},
		{name: "bare underscore prefix stripped", trailer: "x_unkey", stripped: true},
		{name: "underscore prefix extension stripped", trailer: "x_unkeyabc", stripped: true},
		{name: "unkey hyphen prefix stripped", trailer: "Unkey-X", stripped: true},
		{name: "unkey underscore prefix stripped", trailer: "uNkEy_x", stripped: true},
		{name: "unkey without separator stripped", trailer: "Unkeyed", stripped: true},
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
