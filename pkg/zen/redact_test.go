package zen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedact_KeyField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple key value is redacted",
			in:   `{"key": "sk_live_abc123"}`,
			want: `{"key": "[REDACTED]"}`,
		},
		{
			name: "key value with no whitespace around colon",
			in:   `{"key":"sk_live_abc123"}`,
			want: `{"key": "[REDACTED]"}`,
		},
		{
			name: "key value with extra whitespace around colon",
			in:   `{"key"  :  "sk_live_abc123"}`,
			want: `{"key": "[REDACTED]"}`,
		},
		{
			name: "key value with tab whitespace",
			in:   "\"key\"\t:\t\"sk_live_abc123\"",
			want: `"key": "[REDACTED]"`,
		},
		{
			name: "key value with escaped characters",
			in:   `{"key": "value\"with\\escapes"}`,
			want: `{"key": "[REDACTED]"}`,
		},
		{
			name: "key value in full key response body",
			in:   `{"keyId":"key_123","start":"sk_live","key":"sk_live_abcdef123456","enabled":true}`,
			want: `{"keyId":"key_123","start":"sk_live","key": "[REDACTED]","enabled":true}`,
		},
		{
			name: "empty key value is redacted",
			in:   `{"key": ""}`,
			want: `{"key": "[REDACTED]"}`,
		},
		{
			name: "key value with special characters",
			in:   `{"key": "sk_live_!@#$%^&*()"}`,
			want: `{"key": "[REDACTED]"}`,
		},
		{
			name: "key value with unicode",
			in:   `{"key": "sk_live_\u00e9\u00e8"}`,
			want: `{"key": "[REDACTED]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := redact([]byte(tt.in))
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestRedact_PlaintextField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple plaintext value is redacted",
			in:   `{"plaintext": "sk_live_secret_value"}`,
			want: `{"plaintext": "[REDACTED]"}`,
		},
		{
			name: "plaintext value with no whitespace around colon",
			in:   `{"plaintext":"sk_live_secret_value"}`,
			want: `{"plaintext": "[REDACTED]"}`,
		},
		{
			name: "plaintext in key response with decrypt enabled",
			in:   `{"keyId":"key_123","plaintext":"sk_live_abcdef","start":"sk_live","enabled":true}`,
			want: `{"keyId":"key_123","plaintext": "[REDACTED]","start":"sk_live","enabled":true}`,
		},
		{
			name: "plaintext value with escaped characters",
			in:   `{"plaintext": "value\"with\\escapes"}`,
			want: `{"plaintext": "[REDACTED]"}`,
		},
		{
			name: "empty plaintext value is redacted",
			in:   `{"plaintext": ""}`,
			want: `{"plaintext": "[REDACTED]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := redact([]byte(tt.in))
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestRedact_BothFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "both key and plaintext are redacted",
			in:   `{"key": "sk_live_abc123", "plaintext": "sk_live_abc123"}`,
			want: `{"key": "[REDACTED]", "plaintext": "[REDACTED]"}`,
		},
		{
			name: "full create key response body",
			in:   `{"keyId":"key_123","key":"sk_live_newkey","name":"test","plaintext":"sk_live_newkey","enabled":true,"createdAt":1700000000}`,
			want: `{"keyId":"key_123","key": "[REDACTED]","name":"test","plaintext": "[REDACTED]","enabled":true,"createdAt":1700000000}`,
		},
		{
			name: "nested JSON with both fields",
			in:   `{"data":{"key":"secret","plaintext":"also_secret"},"meta":{"requestId":"req_123"}}`,
			want: `{"data":{"key": "[REDACTED]","plaintext": "[REDACTED]"},"meta":{"requestId":"req_123"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := redact([]byte(tt.in))
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestRedact_NonSensitiveFieldsUnchanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{
			name: "error response is unchanged",
			in:   `{"error":{"type":"https://unkey.com/docs/errors","title":"Bad Request","status":400,"detail":"Invalid request"},"meta":{"requestId":"req_123"}}`,
		},
		{
			name: "verify key response without sensitive fields",
			in:   `{"valid":true,"code":"VALID","keyId":"key_123","enabled":true}`,
		},
		{
			name: "ratelimit response is unchanged",
			in:   `{"autoApply":true,"duration":60000,"id":"rl_123","limit":100,"name":"api_requests"}`,
		},
		{
			name: "identity response is unchanged",
			in:   `{"id":"id_123","externalId":"ext_123","meta":{"plan":"pro"}}`,
		},
		{
			name: "empty object is unchanged",
			in:   `{}`,
		},
		{
			name: "empty input is unchanged",
			in:   ``,
		},
		{
			name: "field named apiKey is not redacted",
			in:   `{"apiKey": "some_value"}`,
		},
		{
			name: "field named keyId is not redacted",
			in:   `{"keyId": "key_123"}`,
		},
		{
			name: "plain text without quotes is unchanged",
			in:   `this is just plain text with key and plaintext words`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := redact([]byte(tt.in))
			require.Equal(t, tt.in, string(got))
		})
	}
}

func TestRedact_MultipleOccurrences(t *testing.T) {
	t.Parallel()

	in := `[{"key": "secret1"}, {"key": "secret2"}, {"key": "secret3"}]`
	want := `[{"key": "[REDACTED]"}, {"key": "[REDACTED]"}, {"key": "[REDACTED]"}]`

	got := redact([]byte(in))
	require.Equal(t, want, string(got))
}

func TestRedact_PreservesNonMatchingContent(t *testing.T) {
	t.Parallel()

	in := `{"keyId": "key_123", "key": "secret", "name": "my-key", "meta": {"env": "prod"}}`
	got := string(redact([]byte(in)))

	require.Contains(t, got, `"keyId": "key_123"`)
	require.Contains(t, got, `"key": "[REDACTED]"`)
	require.Contains(t, got, `"name": "my-key"`)
	require.Contains(t, got, `"meta": {"env": "prod"}`)
}

func FuzzRedact(f *testing.F) {
	f.Add([]byte(`{"key": "sk_live_abc123"}`))
	f.Add([]byte(`{"plaintext": "secret"}`))
	f.Add([]byte(`{"key": "a", "plaintext": "b"}`))
	f.Add([]byte(`{"keyId": "key_123"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))
	f.Add([]byte(`not json at all`))
	f.Add([]byte(`{"key": "value\"with\\escapes"}`))

	f.Fuzz(func(t *testing.T, input []byte) {
		result := redact(input)

		if len(input) > 0 {
			require.NotNil(t, result)
		}

		resultStr := string(result)
		require.NotContains(t, resultStr, `"key": "sk_live_abc123"`)
		require.NotContains(t, resultStr, `"plaintext": "secret"`)
	})
}
