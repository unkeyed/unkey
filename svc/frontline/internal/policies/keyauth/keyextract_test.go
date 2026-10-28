package keyauth

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestExtractKey_DefaultBearer(t *testing.T) {
	t.Parallel()

	req := &http.Request{Header: http.Header{}}
	req.Header.Set("Authorization", "Bearer sk_live_abc123")

	key := extractKey(req, nil)
	assert.Equal(t, "sk_live_abc123", key)
}

func TestExtractKey_BearerCaseInsensitive(t *testing.T) {
	t.Parallel()

	req := &http.Request{Header: http.Header{}}
	req.Header.Set("Authorization", "bearer sk_live_abc123")

	key := extractBearer(req)
	assert.Equal(t, "sk_live_abc123", key)
}

func TestExtractKey_NoAuthHeader(t *testing.T) {
	t.Parallel()

	req := &http.Request{Header: http.Header{}}
	key := extractKey(req, nil)
	assert.Empty(t, key)
}

func TestExtractKey_BearerLocation(t *testing.T) {
	t.Parallel()

	req := &http.Request{Header: http.Header{}}
	req.Header.Set("Authorization", "Bearer my_key")

	//nolint:exhaustruct
	locations := []*frontlinev1.KeyLocation{
		{Location: &frontlinev1.KeyLocation_Bearer{Bearer: &frontlinev1.BearerTokenLocation{}}},
	}

	key := extractKey(req, locations)
	assert.Equal(t, "my_key", key)
}

func TestExtractKey_HeaderLocation(t *testing.T) {
	t.Parallel()

	req := &http.Request{Header: http.Header{}}
	req.Header.Set("X-API-Key", "custom_key_123")

	//nolint:exhaustruct
	locations := []*frontlinev1.KeyLocation{
		{Location: &frontlinev1.KeyLocation_Header{
			Header: &frontlinev1.HeaderKeyLocation{Name: "X-API-Key"},
		}},
	}

	key := extractKey(req, locations)
	assert.Equal(t, "custom_key_123", key)
}

func TestExtractKey_HeaderWithStripPrefix(t *testing.T) {
	t.Parallel()

	req := &http.Request{Header: http.Header{}}
	req.Header.Set("Authorization", "ApiKey sk_live_abc123")

	//nolint:exhaustruct
	locations := []*frontlinev1.KeyLocation{
		{Location: &frontlinev1.KeyLocation_Header{
			Header: &frontlinev1.HeaderKeyLocation{
				Name:        "Authorization",
				StripPrefix: "ApiKey ",
			},
		}},
	}

	key := extractKey(req, locations)
	assert.Equal(t, "sk_live_abc123", key)
}

func TestExtractKey_HeaderStripPrefixMismatch(t *testing.T) {
	t.Parallel()

	req := &http.Request{Header: http.Header{}}
	req.Header.Set("Authorization", "Bearer sk_live_abc123")

	//nolint:exhaustruct
	locations := []*frontlinev1.KeyLocation{
		{Location: &frontlinev1.KeyLocation_Header{
			Header: &frontlinev1.HeaderKeyLocation{
				Name:        "Authorization",
				StripPrefix: "ApiKey ",
			},
		}},
	}

	key := extractKey(req, locations)
	assert.Empty(t, key)
}

func TestExtractKey_QueryParam(t *testing.T) {
	t.Parallel()

	req := &http.Request{
		Header: http.Header{},
		URL:    &url.URL{RawQuery: "api_key=query_key_123"},
	}

	//nolint:exhaustruct
	locations := []*frontlinev1.KeyLocation{
		{Location: &frontlinev1.KeyLocation_QueryParam{
			QueryParam: &frontlinev1.QueryParamKeyLocation{Name: "api_key"},
		}},
	}

	key := extractKey(req, locations)
	assert.Equal(t, "query_key_123", key)
}

func TestExtractKey_FallbackOrder(t *testing.T) {
	t.Parallel()

	// First location has nothing, second has the key
	req := &http.Request{
		Header: http.Header{},
		URL:    &url.URL{RawQuery: "token=fallback_key"},
	}

	//nolint:exhaustruct
	locations := []*frontlinev1.KeyLocation{
		{Location: &frontlinev1.KeyLocation_Header{
			Header: &frontlinev1.HeaderKeyLocation{Name: "X-API-Key"},
		}},
		{Location: &frontlinev1.KeyLocation_QueryParam{
			QueryParam: &frontlinev1.QueryParamKeyLocation{Name: "token"},
		}},
	}

	key := extractKey(req, locations)
	assert.Equal(t, "fallback_key", key)
}

func TestExtractKey_FirstLocationWins(t *testing.T) {
	t.Parallel()

	req := &http.Request{
		Header: http.Header{},
		URL:    &url.URL{RawQuery: "token=query_key"},
	}
	req.Header.Set("X-API-Key", "header_key")

	//nolint:exhaustruct
	locations := []*frontlinev1.KeyLocation{
		{Location: &frontlinev1.KeyLocation_Header{
			Header: &frontlinev1.HeaderKeyLocation{Name: "X-API-Key"},
		}},
		{Location: &frontlinev1.KeyLocation_QueryParam{
			QueryParam: &frontlinev1.QueryParamKeyLocation{Name: "token"},
		}},
	}

	key := extractKey(req, locations)
	assert.Equal(t, "header_key", key)
}

func TestRejectRepeatedKey(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	bearer := &frontlinev1.KeyLocation{Location: &frontlinev1.KeyLocation_Bearer{Bearer: &frontlinev1.BearerTokenLocation{}}}
	//nolint:exhaustruct
	header := &frontlinev1.KeyLocation{Location: &frontlinev1.KeyLocation_Header{Header: &frontlinev1.HeaderKeyLocation{Name: "X-API-Key"}}}
	//nolint:exhaustruct
	query := &frontlinev1.KeyLocation{Location: &frontlinev1.KeyLocation_QueryParam{QueryParam: &frontlinev1.QueryParamKeyLocation{Name: "api_key"}}}

	tests := []struct {
		name      string
		header    http.Header
		query     string
		locations []*frontlinev1.KeyLocation
		rejected  bool
	}{
		{"single default bearer", http.Header{"Authorization": {"Bearer a"}}, "", nil, false},
		{"repeated default bearer", http.Header{"Authorization": {"Bearer a", "Bearer b"}}, "", nil, true},
		{"repeated bearer location", http.Header{"Authorization": {"Bearer a", "Bearer b"}}, "", []*frontlinev1.KeyLocation{bearer}, true},
		{"single header", http.Header{"X-Api-Key": {"a"}}, "", []*frontlinev1.KeyLocation{header}, false},
		{"repeated header", http.Header{"X-Api-Key": {"a", "b"}}, "", []*frontlinev1.KeyLocation{header}, true},
		{"single query param", http.Header{}, "api_key=a&other=1&other=2", []*frontlinev1.KeyLocation{query}, false},
		{"repeated query param", http.Header{}, "api_key=a&api_key=b", []*frontlinev1.KeyLocation{query}, true},
		{"repeated in a later location", http.Header{"Authorization": {"Bearer a"}}, "api_key=a&api_key=b", []*frontlinev1.KeyLocation{bearer, query}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := &http.Request{Header: tt.header, URL: &url.URL{RawQuery: tt.query}}
			err := rejectRepeatedKey(req, tt.locations)
			if !tt.rejected {
				require.NoError(t, err)
				return
			}
			urn, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Frontline.Auth.InvalidKey.URN(), urn)
		})
	}
}
