package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateContentEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		encoding string
		data     any
		wantErr  bool
	}{
		{
			name:     "valid base64",
			encoding: "base64",
			data:     "SGVsbG8gV29ybGQh",
			wantErr:  false,
		},
		{
			name:     "invalid base64",
			encoding: "base64",
			data:     "not-valid-base64!!!",
			wantErr:  true,
		},
		{
			name:     "valid base64url",
			encoding: "base64url",
			data:     "SGVsbG8gV29ybGQh",
			wantErr:  false,
		},
		{
			name:     "invalid base64url",
			encoding: "base64url",
			data:     "not-valid!!!",
			wantErr:  true,
		},
		{
			name:     "unknown encoding passes through",
			encoding: "unknown-encoding",
			data:     "any data",
			wantErr:  false,
		},
		{
			name:     "non-string data returns error",
			encoding: "base64",
			data:     123,
			wantErr:  true,
		},
		{
			name:     "empty string is valid base64",
			encoding: "base64",
			data:     "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateContentEncoding(tt.encoding, tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateContentMediaType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mediaType string
		data      any
		wantErr   bool
	}{
		{
			name:      "valid JSON",
			mediaType: "application/json",
			data:      `{"key": "value"}`,
			wantErr:   false,
		},
		{
			name:      "invalid JSON",
			mediaType: "application/json",
			data:      `{invalid json}`,
			wantErr:   true,
		},
		{
			name:      "valid JSON array",
			mediaType: "application/json",
			data:      `[1, 2, 3]`,
			wantErr:   false,
		},
		{
			name:      "valid JSON string",
			mediaType: "application/json",
			data:      `"hello"`,
			wantErr:   false,
		},
		{
			name:      "valid XML (application/xml)",
			mediaType: "application/xml",
			data:      `<root><item>value</item></root>`,
			wantErr:   false,
		},
		{
			name:      "valid XML (text/xml)",
			mediaType: "text/xml",
			data:      `<root><item>value</item></root>`,
			wantErr:   false,
		},
		{
			name:      "invalid XML",
			mediaType: "application/xml",
			data:      `<root><unclosed>`,
			wantErr:   true,
		},
		{
			name:      "unknown media type passes through",
			mediaType: "text/plain",
			data:      "any plain text",
			wantErr:   false,
		},
		{
			name:      "non-string data returns error",
			mediaType: "application/json",
			data:      123,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateContentMediaType(tt.mediaType, tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExtractContentValidators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema   map[string]any
		expected []ContentValidator
	}{
		{
			name: "top-level contentMediaType",
			schema: map[string]any{
				"type":             "string",
				"contentMediaType": "application/json",
			},
			expected: []ContentValidator{
				{Path: "", MediaType: "application/json", Encoding: ""},
			},
		},
		{
			name: "top-level contentEncoding",
			schema: map[string]any{
				"type":            "string",
				"contentEncoding": "base64",
			},
			expected: []ContentValidator{
				{Path: "", MediaType: "", Encoding: "base64"},
			},
		},
		{
			name: "both contentMediaType and contentEncoding",
			schema: map[string]any{
				"type":             "string",
				"contentMediaType": "application/json",
				"contentEncoding":  "base64",
			},
			expected: []ContentValidator{
				{Path: "", MediaType: "application/json", Encoding: "base64"},
			},
		},
		{
			name: "nested property with contentMediaType",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data": map[string]any{
						"type":             "string",
						"contentMediaType": "application/json",
					},
				},
			},
			expected: []ContentValidator{
				{Path: "data", MediaType: "application/json", Encoding: ""},
			},
		},
		{
			name: "deeply nested property",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"outer": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"inner": map[string]any{
								"type":            "string",
								"contentEncoding": "base64",
							},
						},
					},
				},
			},
			expected: []ContentValidator{
				{Path: "outer.inner", MediaType: "", Encoding: "base64"},
			},
		},
		{
			name: "no content validation",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validators := ExtractContentValidators(tt.schema, "")
			require.Equal(t, tt.expected, validators)
		})
	}
}

func TestGetValueAtPath(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"name": "test",
		"nested": map[string]any{
			"value": "inner",
			"deep": map[string]any{
				"item": "deepest",
			},
		},
		"items": []any{"a", "b", "c"},
	}

	tests := []struct {
		name     string
		path     string
		expected any
		found    bool
	}{
		{
			name:     "empty path returns root",
			path:     "",
			expected: data,
			found:    true,
		},
		{
			name:     "top-level property",
			path:     "name",
			expected: "test",
			found:    true,
		},
		{
			name:     "nested property",
			path:     "nested.value",
			expected: "inner",
			found:    true,
		},
		{
			name:     "deeply nested property",
			path:     "nested.deep.item",
			expected: "deepest",
			found:    true,
		},
		{
			name:     "missing property",
			path:     "nonexistent",
			expected: nil,
			found:    false,
		},
		{
			name:     "array property",
			path:     "items",
			expected: []any{"a", "b", "c"},
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, found := GetValueAtPath(data, tt.path)
			require.Equal(t, tt.found, found)
			if found {
				require.Equal(t, tt.expected, result)
			}
		})
	}
}
