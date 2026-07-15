package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestWithReservedHeaderStrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		header   string
		stripped bool
	}{
		{name: "principal stripped", header: "X-Unkey-Principal", stripped: true},
		{name: "request id stripped", header: "X-Unkey-Request-Id", stripped: true},
		{name: "lowercase canonicalized then stripped", header: "x-unkey-principal", stripped: true},
		{name: "unrelated header kept", header: "Authorization", stripped: false},
		{name: "X-Forwarded-For kept", header: "X-Forwarded-For", stripped: false},
		{name: "user-defined X-Unkey-Foo stripped", header: "X-Unkey-Foo", stripped: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(tt.header, "spoofed")

			w := httptest.NewRecorder()
			//nolint:exhaustruct
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))

			var seen http.Header
			next := func(_ context.Context, s *zen.Session) error {
				seen = s.Request().Header.Clone()
				return nil
			}

			err := WithReservedHeaderStrip()(next)(context.Background(), sess)
			require.NoError(t, err)

			canonical := http.CanonicalHeaderKey(tt.header)
			if tt.stripped {
				require.Empty(t, seen.Get(canonical), "%q should be stripped", canonical)
			} else {
				require.Equal(t, "spoofed", seen.Get(canonical), "%q should be preserved", canonical)
			}
		})
	}
}
