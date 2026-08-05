package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/redaction"
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

func TestRedactBody_AppliesOpenAPIRedactors(t *testing.T) {
	body := []byte(`{"secret":"hide-me","visible":"keep-me"}`)

	got := redactBody(body, []*redaction.Redactor{redaction.New([]string{"secret"})})

	require.Equal(t, `{"secret":"[REDACTED]","visible":"keep-me"}`, got)
	require.Equal(t, `{"secret":"hide-me","visible":"keep-me"}`, string(body))
}

var benchmarkRedactedBody string

func BenchmarkRedactBody(b *testing.B) {
	body := []byte(`{"message":"` + strings.Repeat("a", 1024) + `","token":"hide-me","visible":"keep-me"}`)
	tests := []struct {
		name      string
		redactors []*redaction.Redactor
	}{
		{name: "before", redactors: nil},
		{name: "redaction_on", redactors: []*redaction.Redactor{redaction.New([]string{"token"})}},
		{name: "redaction_off", redactors: nil},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			for b.Loop() {
				benchmarkRedactedBody = redactBody(body, test.redactors)
			}
		})
	}
}
