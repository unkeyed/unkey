package middleware

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatHeaders_RedactsAuthorization(t *testing.T) {
	h := http.Header{
		"Authorization": []string{"Bearer sk_live_secret"},
		"Content-Type":  []string{"application/json"},
	}

	got := formatHeaders(h, nil)

	require.Contains(t, got, "Authorization: [REDACTED]")
	require.Contains(t, got, "Content-Type: application/json")
	require.NotContains(t, got, "sk_live_secret")
}

func TestFormatHeaders_RedactsConfiguredSecretHeaders(t *testing.T) {
	h := http.Header{
		"X-Api-Key":    []string{"sk_live_secret"},
		"Content-Type": []string{"application/json"},
	}

	// Secret names are lowercased, matching http.Header's canonicalized keys
	// after ToLower.
	got := formatHeaders(h, map[string]struct{}{"x-api-key": {}})

	require.Contains(t, got, "X-Api-Key: [REDACTED]")
	require.Contains(t, got, "Content-Type: application/json")
	require.NotContains(t, got, "sk_live_secret")
}

func TestFormatHeaders_NoSecretsLeavesValues(t *testing.T) {
	h := http.Header{"X-Api-Key": []string{"sk_live_secret"}}

	got := formatHeaders(h, nil)

	require.Contains(t, got, "X-Api-Key: sk_live_secret")
}

func TestRedactQueryParams_RedactsConfigured(t *testing.T) {
	values := url.Values{
		"api_key": []string{"sk_live_secret"},
		"page":    []string{"2"},
	}

	got := redactQueryParams(values, map[string]struct{}{"api_key": {}})

	require.Equal(t, []string{"[REDACTED]"}, got["api_key"])
	require.Equal(t, []string{"2"}, got["page"])
	// Input is not mutated.
	require.Equal(t, []string{"sk_live_secret"}, values["api_key"])
}

func TestToSet(t *testing.T) {
	require.Nil(t, toSet(nil))
	require.Nil(t, toSet([]string{}))

	set := toSet([]string{"a", "b"})
	require.Contains(t, set, "a")
	require.Contains(t, set, "b")
}
