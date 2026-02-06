package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCompilerCleanForJSONSchema tests the cleanForJSONSchema function
func TestCompilerCleanForJSONSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "converts nullable:true with string type to type array",
			input: map[string]any{
				"type":     "string",
				"nullable": true,
			},
			expected: map[string]any{
				"type": []any{"string", "null"},
			},
		},
		{
			name: "nullable:false is removed without type change",
			input: map[string]any{
				"type":     "string",
				"nullable": false,
			},
			expected: map[string]any{
				"type": "string",
			},
		},
		{
			name: "nullable:true appends null to existing type array",
			input: map[string]any{
				"type":     []any{"string", "integer"},
				"nullable": true,
			},
			expected: map[string]any{
				"type": []any{"string", "integer", "null"},
			},
		},
		{
			name: "nullable:true with type array already containing null",
			input: map[string]any{
				"type":     []any{"string", "null"},
				"nullable": true,
			},
			expected: map[string]any{
				"type": []any{"string", "null"},
			},
		},
		{
			name: "nullable:true without type adds null type",
			input: map[string]any{
				"nullable": true,
			},
			expected: map[string]any{
				"type": []any{"null"},
			},
		},
		{
			name: "removes readOnly/writeOnly",
			input: map[string]any{
				"type":      "string",
				"readOnly":  true,
				"writeOnly": false,
			},
			expected: map[string]any{
				"type": "string",
			},
		},
		{
			name: "removes deprecated",
			input: map[string]any{
				"type":       "string",
				"deprecated": true,
			},
			expected: map[string]any{
				"type": "string",
			},
		},
		{
			name: "fixes null in type array",
			input: map[string]any{
				"type": []any{"string", nil},
			},
			expected: map[string]any{
				"type": []any{"string", "null"},
			},
		},
		{
			name: "recursively cleans nested objects with nullable conversion",
			input: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":     "string",
						"nullable": true,
					},
				},
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": []any{"string", "null"},
					},
				},
			},
		},
		{
			name: "converts nullable:true with integer type",
			input: map[string]any{
				"type":     "integer",
				"nullable": true,
			},
			expected: map[string]any{
				"type": []any{"integer", "null"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cleanForJSONSchema(tt.input)
			resultMap, ok := result.(map[string]any)
			require.True(t, ok, "result should be a map")

			// Compare key by key since map ordering can vary
			require.Equal(t, len(tt.expected), len(resultMap), "map lengths should match")
			for k, v := range tt.expected {
				require.Contains(t, resultMap, k, "result should contain key %s", k)
				require.Equal(t, v, resultMap[k], "values for key %s should match", k)
			}
		})
	}
}
