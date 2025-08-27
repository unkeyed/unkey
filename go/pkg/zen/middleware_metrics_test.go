package zen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedact(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redacts key field",
			input:    `{"key": "sk_live_1234567890abcdef"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "redacts plaintext field",
			input:    `{"plaintext": "super secret data"}`,
			expected: `{"plaintext": "[REDACTED]"}`,
		},
		{
			name:     "redacts multiple fields",
			input:    `{"key": "sk_live_xyz", "plaintext": "sensitive", "other": "visible"}`,
			expected: `{"key": "[REDACTED]", "plaintext": "[REDACTED]", "other": "visible"}`,
		},
		{
			name:     "redacts key with no spaces after colon",
			input:    `{"key":"sk_test_123"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "redacts key with multiple spaces",
			input:    `{"key":     "sk_test_123"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "redacts key with tabs",
			input:    `{"key":	"sk_test_123"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "redacts key with newline",
			input:    `{"key":
"sk_test_123"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "redacts plaintext with no spaces",
			input:    `{"plaintext":"secret1"}`,
			expected: `{"plaintext": "[REDACTED]"}`,
		},
		{
			name:     "redacts plaintext with multiple spaces",
			input:    `{"plaintext":    "secret2"}`,
			expected: `{"plaintext": "[REDACTED]"}`,
		},
		{
			name:     "redacts plaintext with tabs",
			input:    `{"plaintext":	"secret3"}`,
			expected: `{"plaintext": "[REDACTED]"}`,
		},
		{
			name:     "handles nested JSON",
			input:    `{"data": {"key": "sk_live_nested", "value": 123}}`,
			expected: `{"data": {"key": "[REDACTED]", "value": 123}}`,
		},
		{
			name:     "handles arrays with sensitive data",
			input:    `{"items": [{"key": "sk1"}, {"key": "sk2"}]}`,
			expected: `{"items": [{"key": "[REDACTED]"}, {"key": "[REDACTED]"}]}`,
		},
		{
			name:     "preserves non-sensitive data",
			input:    `{"id": 1, "name": "test", "status": "active"}`,
			expected: `{"id": 1, "name": "test", "status": "active"}`,
		},
		{
			name:     "handles empty sensitive fields",
			input:    `{"key": "", "plaintext": ""}`,
			expected: `{"key": "[REDACTED]", "plaintext": "[REDACTED]"}`,
		},
		{
			name:     "handles special characters in sensitive fields",
			input:    `{"key": "sk_live_!@#$%^&*()", "plaintext": "data\nwith\nnewlines"}`,
			expected: `{"key": "[REDACTED]", "plaintext": "[REDACTED]"}`,
		},
		{
			name:     "handles multiple occurrences in single line",
			input:    `{"key": "sk1", "data": {"key": "sk2", "nested": {"key": "sk3"}}}`,
			expected: `{"key": "[REDACTED]", "data": {"key": "[REDACTED]", "nested": {"key": "[REDACTED]"}}}`,
		},
		{
			name:     "handles mixed sensitive and non-sensitive fields",
			input:    `{"key": "secret", "keyName": "not_secret", "plaintext": "hidden", "plain": "visible"}`,
			expected: `{"key": "[REDACTED]", "keyName": "not_secret", "plaintext": "[REDACTED]", "plain": "visible"}`,
		},
		{
			name:     "handles malformed JSON gracefully",
			input:    `not valid json but has "key": "value" and "plaintext": "data"`,
			expected: `not valid json but has "key": "[REDACTED]" and "plaintext": "[REDACTED]"`,
		},
		{
			name:     "handles multiline JSON",
			input:    "{\n  \"key\": \"sk_live_multiline\",\n  \"plaintext\": \"multiline_secret\"\n}",
			expected: "{\n  \"key\": \"[REDACTED]\",\n  \"plaintext\": \"[REDACTED]\"\n}",
		},
		{
			name:     "handles URL encoded sensitive data",
			input:    `{"key": "sk%20live%20encoded", "plaintext": "secret%20data"}`,
			expected: `{"key": "[REDACTED]", "plaintext": "[REDACTED]"}`,
		},
		{
			name:     "redacts only exact field matches",
			input:    `{"apikey": "visible", "key": "hidden", "plaintextual": "visible", "plaintext": "hidden"}`,
			expected: `{"apikey": "visible", "key": "[REDACTED]", "plaintextual": "visible", "plaintext": "[REDACTED]"}`,
		},
		{
			name:     "handles very long sensitive values",
			input:    `{"key": "` + string(make([]byte, 1000)) + `"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "handles Unicode in sensitive fields",
			input:    `{"key": "sk_live_你好世界", "plaintext": "🔐🔑🗝️"}`,
			expected: `{"key": "[REDACTED]", "plaintext": "[REDACTED]"}`,
		},
		{
			name:     "handles empty input",
			input:    ``,
			expected: ``,
		},
		{
			name:     "handles input with only whitespace",
			input:    `   `,
			expected: `   `,
		},
		{
			name:     "handles complex nested structure",
			input: `{
				"request": {
					"headers": {"authorization": "Bearer token"},
					"body": {"key": "sk_live_xyz", "plaintext": "secret_data"}
				},
				"response": {
					"data": {"key": "response_key", "plaintext": "response_secret"}
				}
			}`,
			expected: `{
				"request": {
					"headers": {"authorization": "Bearer token"},
					"body": {"key": "[REDACTED]", "plaintext": "[REDACTED]"}
				},
				"response": {
					"data": {"key": "[REDACTED]", "plaintext": "[REDACTED]"}
				}
			}`,
		},
		{
			name:     "handles arrays of sensitive data",
			input:    `{"keys": ["key": "sk1", "key": "sk2"], "plaintexts": ["plaintext": "s1", "plaintext": "s2"]}`,
			expected: `{"keys": ["key": "[REDACTED]", "key": "[REDACTED]"], "plaintexts": ["plaintext": "[REDACTED]", "plaintext": "[REDACTED]"]}`,
		},
		{
			name:     "handles text before sensitive field",
			input:    `some text before "key": "secret" and after`,
			expected: `some text before "key": "[REDACTED]" and after`,
		},
		{
			name:     "handles multiple fields on same line",
			input:    `{"key": "secret1", "key": "secret2", "key": "secret3"}`,
			expected: `{"key": "[REDACTED]", "key": "[REDACTED]", "key": "[REDACTED]"}`,
		},
		{
			name:     "handles key field with prefix text",
			input:    `prefix text "key": "value" suffix`,
			expected: `prefix text "key": "[REDACTED]" suffix`,
		},
		{
			name:     "handles plaintext field with prefix text",
			input:    `some data "plaintext": "secret" more data`,
			expected: `some data "plaintext": "[REDACTED]" more data`,
		},
		{
			name:     "handles fields in query params",
			input:    `https://api.com?data={"key": "sk_live_123"}&other=value`,
			expected: `https://api.com?data={"key": "[REDACTED]"}&other=value`,
		},
		{
			name:     "handles fields in log messages",
			input:    `[ERROR] Failed to process request with "key": "sk_test_abc" at timestamp`,
			expected: `[ERROR] Failed to process request with "key": "[REDACTED]" at timestamp`,
		},
		{
			name:     "handles fields with mixed quotes",
			input:    `data with "key": "secret" and 'other': 'value'`,
			expected: `data with "key": "[REDACTED]" and 'other': 'value'`,
		},
		{
			name:     "handles consecutive sensitive fields",
			input:    `"key": "val1""key": "val2""key": "val3"`,
			expected: `"key": "[REDACTED]""key": "[REDACTED]""key": "[REDACTED]"`,
		},
		{
			name:     "handles fields with various whitespace combinations",
			input:    `"key" : "val1", "key"  :  "val2", "key"	:	"val3"`,
			expected: `"key": "[REDACTED]", "key": "[REDACTED]", "key": "[REDACTED]"`,
		},
		{
			name:     "handles fields at start of string",
			input:    `"key": "secret" rest of content`,
			expected: `"key": "[REDACTED]" rest of content`,
		},
		{
			name:     "handles fields at end of string",
			input:    `content before "plaintext": "secret"`,
			expected: `content before "plaintext": "[REDACTED]"`,
		},
		{
			name:     "handles embedded JSON in string",
			input:    `log: {"level": "error", "key": "sk_123", "msg": "failed"}`,
			expected: `log: {"level": "error", "key": "[REDACTED]", "msg": "failed"}`,
		},
		{
			name:     "handles XML-like content with sensitive JSON fields",
			input:    `<data>{"key": "secret"}</data>`,
			expected: `<data>{"key": "[REDACTED]"}</data>`,
		},
		{
			name:     "handles markdown with sensitive fields",
			input:    "```json\n{\"key\": \"secret\"}\n```",
			expected: "```json\n{\"key\": \"[REDACTED]\"}\n```",
		},
		{
			name:     "handles sensitive fields in error stack traces",
			input:    `Error: Invalid key\n  at validateKey ({"key": "sk_test"})\n  at line 42`,
			expected: `Error: Invalid key\n  at validateKey ({"key": "[REDACTED]"})\n  at line 42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redact([]byte(tt.input))
			require.Equal(t, tt.expected, string(result))
		})
	}
}

func TestRedactBinaryData(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "handles binary data with embedded sensitive JSON",
			input:    []byte{0x00, 0x01, '"', 'k', 'e', 'y', '"', ':', ' ', '"', 's', 'e', 'c', 'r', 'e', 't', '"', 0xFF},
			expected: []byte{0x00, 0x01, '"', 'k', 'e', 'y', '"', ':', ' ', '"', '[', 'R', 'E', 'D', 'A', 'C', 'T', 'E', 'D', ']', '"', 0xFF},
		},
		{
			name:     "handles null bytes in input",
			input:    []byte("before\x00{\"key\": \"secret\"}\x00after"),
			expected: []byte("before\x00{\"key\": \"[REDACTED]\"}\x00after"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redact(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactPerformance(t *testing.T) {
	largeInput := `{"key": "sk_live_test", "plaintext": "secret", "data": "` + string(make([]byte, 10000)) + `"}`
	expected := `{"key": "[REDACTED]", "plaintext": "[REDACTED]", "data": "` + string(make([]byte, 10000)) + `"}`

	result := redact([]byte(largeInput))
	require.Equal(t, expected, string(result))
}

func TestRedactIdempotence(t *testing.T) {
	input := `{"key": "secret", "plaintext": "data"}`
	expected := `{"key": "[REDACTED]", "plaintext": "[REDACTED]"}`

	firstPass := redact([]byte(input))
	require.Equal(t, expected, string(firstPass))

	secondPass := redact(firstPass)
	require.Equal(t, expected, string(secondPass), "Redacting already redacted data should not change it")
}

func TestRedactConcurrency(t *testing.T) {
	input := `{"key": "sk_live_concurrent", "plaintext": "concurrent_secret"}`
	expected := `{"key": "[REDACTED]", "plaintext": "[REDACTED]"}`

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			result := redact([]byte(input))
			require.Equal(t, expected, string(result))
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRedactEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "handles escaped backslashes before quote",
			input:    `{"key": "value\\\"with\\\"escapes"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "handles value with only quotes",
			input:    `{"key": "\"\"\""}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "handles single quote inside double quoted value",
			input:    `{"key": "val'ue", "plaintext": "sec'ret"}`,
			expected: `{"key": "[REDACTED]", "plaintext": "[REDACTED]"}`,
		},
		{
			name:     "handles regex special chars in value",
			input:    `{"key": ".*+?[]{}()|\\^$"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "handles value that looks like JSON",
			input:    `{"key": "{\"nested\": \"json\"}"}`,
			expected: `{"key": "[REDACTED]"}`,
		},
		{
			name:     "handles multiple whitespace types together",
			input:    "\"key\" \t \n : \t \n \"secret\"",
			expected: `"key": "[REDACTED]"`,
		},
		{
			name:     "redacts when field appears multiple times with different spacing",
			input:    `"key":"val1", "key" :"val2", "key" : "val3", "key"  :  "val4"`,
			expected: `"key": "[REDACTED]", "key": "[REDACTED]", "key": "[REDACTED]", "key": "[REDACTED]"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redact([]byte(tt.input))
			require.Equal(t, tt.expected, string(result))
		})
	}
}