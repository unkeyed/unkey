package validation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactJSON(t *testing.T) {
	redactions := map[string]*RedactionConfig{
		"POST /v2/keys.createKey": {
			Request: NewRedactionTree([][]string{{"key"}}),
			Response: NewRedactionTree([][]string{
				{"data", "key"},
			}),
		},
		"POST /v2/keys.verifyKey": {
			Request: NewRedactionTree([][]string{{"key"}}),
		},
		"GET /v2/keys.getKey": {
			Response: NewRedactionTree([][]string{
				{"data", "plaintext"},
			}),
		},
	}

	tests := []struct {
		name       string
		routeKey   string
		isResponse bool
		input      string
		expected   string
	}{
		{
			name:       "redacts response key field at correct path",
			routeKey:   "POST /v2/keys.createKey",
			isResponse: true,
			input:      `{"data":{"key":"sk_live_1234","keyId":"key_abc"},"meta":{"requestId":"req_1"}}`,
			expected:   `{"data":{"key":"[REDACTED]","keyId":"key_abc"},"meta":{"requestId":"req_1"}}`,
		},
		{
			name:       "does not redact key in wrong path",
			routeKey:   "POST /v2/keys.createKey",
			isResponse: true,
			input:      `{"meta":{"key":"not_sensitive"},"data":{"keyId":"key_abc"}}`,
			expected:   `{"data":{"keyId":"key_abc"},"meta":{"key":"not_sensitive"}}`,
		},
		{
			name:       "redacts request key field",
			routeKey:   "POST /v2/keys.verifyKey",
			isResponse: false,
			input:      `{"key":"sk_live_secret","tags":["test"]}`,
			expected:   `{"key":"[REDACTED]","tags":["test"]}`,
		},
		{
			name:       "redacts plaintext in getKey response",
			routeKey:   "GET /v2/keys.getKey",
			isResponse: true,
			input:      `{"data":{"plaintext":"sk_live_xyz","keyId":"key_1"}}`,
			expected:   `{"data":{"keyId":"key_1","plaintext":"[REDACTED]"}}`,
		},
		{
			name:       "compacts JSON when no redaction rules match",
			routeKey:   "GET /v2/apis.getApi",
			isResponse: true,
			input:      `{  "data":  {  "id": "api_1"  }  }`,
			expected:   `{"data":{"id":"api_1"}}`,
		},
		{
			name:       "compacts JSON for route with rules but no fields to redact",
			routeKey:   "POST /v2/keys.createKey",
			isResponse: false,
			input:      `{  "apiId":  "api_123"  }`,
			expected:   `{"apiId":"api_123"}`,
		},
		{
			name:       "returns invalid JSON unchanged",
			routeKey:   "POST /v2/keys.createKey",
			isResponse: true,
			input:      `not json`,
			expected:   `not json`,
		},
		{
			name:       "handles empty input",
			routeKey:   "POST /v2/keys.createKey",
			isResponse: true,
			input:      ``,
			expected:   ``,
		},
		{
			name:       "idempotent on already-redacted data",
			routeKey:   "POST /v2/keys.verifyKey",
			isResponse: false,
			input:      `{"key":"[REDACTED]"}`,
			expected:   `{"key":"[REDACTED]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactJSON(redactions, tt.routeKey, tt.isResponse, []byte(tt.input))
			require.Equal(t, tt.expected, string(result))
		})
	}
}

func TestNewRedactionTree(t *testing.T) {
	t.Run("builds simple tree", func(t *testing.T) {
		tree := NewRedactionTree([][]string{{"key"}})
		require.True(t, tree.Children["key"].Redact)
	})

	t.Run("builds nested tree", func(t *testing.T) {
		tree := NewRedactionTree([][]string{{"data", "key"}})
		require.False(t, tree.Children["data"].Redact)
		require.True(t, tree.Children["data"].Children["key"].Redact)
	})

	t.Run("builds array tree", func(t *testing.T) {
		tree := NewRedactionTree([][]string{{"items", "[]", "key"}})
		require.NotNil(t, tree.Children["items"].Items)
		require.True(t, tree.Children["items"].Items.Children["key"].Redact)
	})

	t.Run("merges multiple paths", func(t *testing.T) {
		tree := NewRedactionTree([][]string{
			{"data", "key"},
			{"data", "plaintext"},
		})
		require.True(t, tree.Children["data"].Children["key"].Redact)
		require.True(t, tree.Children["data"].Children["plaintext"].Redact)
	})
}

func TestRedactWalk(t *testing.T) {
	t.Run("redacts array items", func(t *testing.T) {
		node := NewRedactionTree([][]string{{"items", "[]", "key"}})
		v := map[string]any{
			"items": []any{
				map[string]any{"key": "secret1", "id": 1},
				map[string]any{"key": "secret2", "id": 2},
			},
		}
		redactWalk(v, node)
		items := v["items"].([]any)
		require.Equal(t, "[REDACTED]", items[0].(map[string]any)["key"])
		require.Equal(t, "[REDACTED]", items[1].(map[string]any)["key"])
		require.Equal(t, 1, items[0].(map[string]any)["id"])
	})

	t.Run("ignores non-matching fields", func(t *testing.T) {
		node := NewRedactionTree([][]string{{"secret"}})
		v := map[string]any{
			"public": "visible",
			"secret": "hidden",
		}
		redactWalk(v, node)
		require.Equal(t, "visible", v["public"])
		require.Equal(t, "[REDACTED]", v["secret"])
	})
}

func TestSanitizeRequest_Integration(t *testing.T) {
	t.Parallel()
	v, err := New()
	require.NoError(t, err)

	t.Run("redacts key in verifyKey request body", func(t *testing.T) {
		input := `{"key":"sk_live_secret456"}`
		r := httptest.NewRequest(http.MethodPost, "/v2/keys.verifyKey", nil)
		body, _ := v.SanitizeRequest(r, []byte(input), r.Header)
		require.Contains(t, body, `"[REDACTED]"`)
		require.NotContains(t, body, "sk_live_secret456")
	})

	t.Run("compacts JSON for routes without redaction rules", func(t *testing.T) {
		input := `{  "name":  "test"  }`
		r := httptest.NewRequest(http.MethodGet, "/v2/liveness", nil)
		body, _ := v.SanitizeRequest(r, []byte(input), r.Header)
		require.Equal(t, `{"name":"test"}`, body)
	})

	t.Run("filters infra headers", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/v2/keys.verifyKey", nil)
		r.Header.Set("X-Forwarded-For", "1.2.3.4")
		r.Header.Set("Content-Type", "application/json")
		_, headers := v.SanitizeRequest(r, nil, r.Header)
		for _, h := range headers {
			require.NotContains(t, h, "X-Forwarded-For")
		}
		found := false
		for _, h := range headers {
			if h == "Content-Type: application/json" {
				found = true
			}
		}
		require.True(t, found, "Content-Type should be present")
	})

	t.Run("redacts authorization header via security scheme x-unkey-redact", func(t *testing.T) {
		// createKey requires rootKey auth which has x-unkey-redact: true
		r := httptest.NewRequest(http.MethodPost, "/v2/keys.createKey", nil)
		r.Header.Set("Authorization", "Bearer unkey_secret123")
		r.Header.Set("Content-Type", "application/json")
		_, headers := v.SanitizeRequest(r, nil, r.Header)
		foundAuth := false
		for _, h := range headers {
			if h == "Authorization: [REDACTED]" {
				foundAuth = true
			}
			// Must not contain the actual token
			require.NotContains(t, h, "unkey_secret123")
		}
		require.True(t, foundAuth, "Authorization header should be redacted")
	})
}

func TestSanitizeResponse_Integration(t *testing.T) {
	t.Parallel()
	v, err := New()
	require.NoError(t, err)

	t.Run("redacts key in createKey response", func(t *testing.T) {
		input := `{"meta":{"requestId":"req_1"},"data":{"keyId":"key_abc","key":"sk_live_secret123"}}`
		r := httptest.NewRequest(http.MethodPost, "/v2/keys.createKey", nil)
		body, _ := v.SanitizeResponse(r, []byte(input), http.Header{})
		require.Contains(t, body, `"[REDACTED]"`)
		require.NotContains(t, body, "sk_live_secret123")
		require.Contains(t, body, "key_abc")
	})

	t.Run("redacts key in rerollKey response", func(t *testing.T) {
		input := `{"meta":{"requestId":"req_2"},"data":{"keyId":"key_xyz","key":"sk_live_rerolled"}}`
		r := httptest.NewRequest(http.MethodPost, "/v2/keys.rerollKey", nil)
		body, _ := v.SanitizeResponse(r, []byte(input), http.Header{})
		require.Contains(t, body, `"[REDACTED]"`)
		require.NotContains(t, body, "sk_live_rerolled")
		require.Contains(t, body, "key_xyz")
	})

	t.Run("compacts JSON for routes without redaction rules", func(t *testing.T) {
		input := `{  "name":  "test"  }`
		r := httptest.NewRequest(http.MethodGet, "/v2/liveness", nil)
		body, _ := v.SanitizeResponse(r, []byte(input), http.Header{})
		require.Equal(t, `{"name":"test"}`, body)
	})
}

func TestSanitizeHeaders(t *testing.T) {
	t.Run("filters infra headers and redacts specified ones", func(t *testing.T) {
		headers := http.Header{
			"X-Forwarded-For": []string{"1.2.3.4"},
			"Authorization":   []string{"Bearer secret"},
			"Content-Type":    []string{"application/json"},
			"X-Amzn-Trace-Id": []string{"trace"},
		}
		redact := map[string]bool{"authorization": true}
		result := SanitizeHeaders(headers, redact)

		foundAuth := false
		foundCT := false
		for _, h := range result {
			require.NotContains(t, h, "X-Forwarded-For")
			require.NotContains(t, h, "X-Amzn-Trace-Id")
			if h == "Authorization: [REDACTED]" {
				foundAuth = true
			}
			if h == "Content-Type: application/json" {
				foundCT = true
			}
		}
		require.True(t, foundAuth)
		require.True(t, foundCT)
	})

	t.Run("nil redact set passes all non-infra headers through", func(t *testing.T) {
		headers := http.Header{
			"Authorization": []string{"Bearer token"},
			"Content-Type":  []string{"application/json"},
		}
		result := SanitizeHeaders(headers, nil)
		foundAuth := false
		for _, h := range result {
			if h == "Authorization: Bearer token" {
				foundAuth = true
			}
		}
		require.True(t, foundAuth, "Authorization should NOT be redacted with nil redact set")
	})
}

func TestCompactJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "compacts whitespace",
			input:    `{  "key" :  "value"  }`,
			expected: `{"key":"value"}`,
		},
		{
			name:     "compacts newlines",
			input:    "{\n  \"key\": \"value\"\n}",
			expected: `{"key":"value"}`,
		},
		{
			name:     "already compact",
			input:    `{"key":"value"}`,
			expected: `{"key":"value"}`,
		},
		{
			name:     "returns invalid JSON unchanged",
			input:    `not json`,
			expected: `not json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompactJSON([]byte(tt.input))
			require.Equal(t, tt.expected, string(result))
		})
	}
}
