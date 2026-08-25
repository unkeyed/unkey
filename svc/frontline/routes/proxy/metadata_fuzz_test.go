package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

func FuzzRequestHops(f *testing.F) {
	fuzz.Seed(f)

	codec, err := meta.New("shared-signing-key")
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		headerValue := c.String()

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		req.Header.Set(proxy.HeaderFrontlineMeta, headerValue)

		hops, err := requestHops(req, codec)
		require.NoError(t, err)
		require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))

		if _, unmarshalErr := codec.Unmarshal(headerValue); unmarshalErr != nil {
			require.Empty(t, hops)
		}
	})
}
