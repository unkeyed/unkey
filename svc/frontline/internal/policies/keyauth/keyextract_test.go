package keyauth

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
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
