package resulttransformer

import (
	"context"
	"fmt"
	"testing"
)

func TestExtractStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "plain string",
			input:    "test_value",
			expected: "test_value",
		},
		{
			name:     "ch.Dynamic with braces",
			input:    mockDynamic{value: "id_123"},
			expected: "id_123",
		},
		{
			name:     "ch.Dynamic with braces and spaces",
			input:    mockDynamic{value: "ks_abc"},
			expected: "ks_abc",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "ch.Dynamic empty",
			input:    mockDynamic{value: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStringValue(tt.input)
			if result != tt.expected {
				t.Errorf("extractStringValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTransformWithMappings(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		columnConfigs   []ColumnConfig
		results         []map[string]any
		mappings        []ColumnMapping
		expectedResults []map[string]any
		expectError     bool
	}{
		{
			name: "transform key_space_id to api_id",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "key_space_id",
					VirtualColumn: "apiId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						// Mock resolver: key_space_id -> api_id
						mapping := make(map[string]string)
						for _, id := range actualIDs {
							mapping[id] = "api_" + id[3:] // Convert ks_xxx to api_xxx
						}
						return mapping, nil
					},
				},
			},
			results: []map[string]any{
				{"a": mockDynamic{value: "ks_123"}},
				{"a": mockDynamic{value: "ks_456"}},
			},
			mappings: []ColumnMapping{
				{ResultColumn: "a", ActualColumn: "key_space_id"},
			},
			expectedResults: []map[string]any{
				{"a": "api_123"},
				{"a": "api_456"},
			},
		},
		{
			name: "transform identity_id to external_id",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "identity_id",
					VirtualColumn: "externalId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						// Mock resolver: identity_id -> external_id
						mapping := make(map[string]string)
						for _, id := range actualIDs {
							mapping[id] = "user_" + id[3:] // Convert id_xxx to user_xxx
						}
						return mapping, nil
					},
				},
			},
			results: []map[string]any{
				{"c": mockDynamic{value: "id_abc"}},
				{"c": mockDynamic{value: "id_def"}},
			},
			mappings: []ColumnMapping{
				{ResultColumn: "c", ActualColumn: "identity_id"},
			},
			expectedResults: []map[string]any{
				{"c": "user_abc"},
				{"c": "user_def"},
			},
		},
		{
			name: "transform multiple columns in same row",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "key_space_id",
					VirtualColumn: "apiId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						mapping := make(map[string]string)
						for _, id := range actualIDs {
							mapping[id] = "api_" + id[3:]
						}
						return mapping, nil
					},
				},
				{
					ActualColumn:  "identity_id",
					VirtualColumn: "externalId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						mapping := make(map[string]string)
						for _, id := range actualIDs {
							mapping[id] = "user_" + id[3:]
						}
						return mapping, nil
					},
				},
			},
			results: []map[string]any{
				{
					"a": mockDynamic{value: "ks_123"},
					"c": mockDynamic{value: "id_abc"},
				},
				{
					"a": mockDynamic{value: "ks_456"},
					"c": mockDynamic{value: "id_def"},
				},
			},
			mappings: []ColumnMapping{
				{ResultColumn: "a", ActualColumn: "key_space_id"},
				{ResultColumn: "c", ActualColumn: "identity_id"},
			},
			expectedResults: []map[string]any{
				{"a": "api_123", "c": "user_abc"},
				{"a": "api_456", "c": "user_def"},
			},
		},
		{
			name: "preserve column aliases",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "key_space_id",
					VirtualColumn: "apiId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						mapping := make(map[string]string)
						for _, id := range actualIDs {
							mapping[id] = "api_" + id[3:]
						}
						return mapping, nil
					},
				},
			},
			results: []map[string]any{
				{"my_custom_alias": mockDynamic{value: "ks_123"}},
			},
			mappings: []ColumnMapping{
				{ResultColumn: "my_custom_alias", ActualColumn: "key_space_id"},
			},
			expectedResults: []map[string]any{
				{"my_custom_alias": "api_123"},
			},
		},
		{
			name: "handle columns with no transformation needed",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "key_space_id",
					VirtualColumn: "apiId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						mapping := make(map[string]string)
						for _, id := range actualIDs {
							mapping[id] = "api_" + id[3:]
						}
						return mapping, nil
					},
				},
			},
			results: []map[string]any{
				{
					"a":       mockDynamic{value: "ks_123"},
					"outcome": mockDynamic{value: "VALID"},
					"count":   mockDynamic{value: "42"},
				},
			},
			mappings: []ColumnMapping{
				{ResultColumn: "a", ActualColumn: "key_space_id"},
			},
			expectedResults: []map[string]any{
				{
					"a":       "api_123",
					"outcome": mockDynamic{value: "VALID"},
					"count":   mockDynamic{value: "42"},
				},
			},
		},
		{
			name: "handle empty results",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "key_space_id",
					VirtualColumn: "apiId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						return make(map[string]string), nil
					},
				},
			},
			results:         []map[string]any{},
			mappings:        []ColumnMapping{},
			expectedResults: []map[string]any{},
		},
		{
			name: "handle resolver errors",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "key_space_id",
					VirtualColumn: "apiId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						return nil, fmt.Errorf("database connection failed")
					},
				},
			},
			results: []map[string]any{
				{"a": mockDynamic{value: "ks_123"}},
			},
			mappings: []ColumnMapping{
				{ResultColumn: "a", ActualColumn: "key_space_id"},
			},
			expectError: true,
		},
		{
			name: "handle IDs not found in resolver mapping",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "key_space_id",
					VirtualColumn: "apiId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						// Return empty mapping - ID not found
						return make(map[string]string), nil
					},
				},
			},
			results: []map[string]any{
				{"a": mockDynamic{value: "ks_notfound"}},
			},
			mappings: []ColumnMapping{
				{ResultColumn: "a", ActualColumn: "key_space_id"},
			},
			expectedResults: []map[string]any{
				// Value should remain unchanged (as ch.Dynamic)
				{"a": mockDynamic{value: "ks_notfound"}},
			},
		},
		{
			name: "batch resolve multiple unique IDs",
			columnConfigs: []ColumnConfig{
				{
					ActualColumn:  "key_space_id",
					VirtualColumn: "apiId",
					Resolver: func(ctx context.Context, actualIDs []string) (map[string]string, error) {
						// Verify we're getting the right IDs
						if len(actualIDs) != 3 {
							return nil, fmt.Errorf("expected 3 IDs, got %d", len(actualIDs))
						}
						mapping := make(map[string]string)
						for _, id := range actualIDs {
							mapping[id] = "api_" + id[3:]
						}
						return mapping, nil
					},
				},
			},
			results: []map[string]any{
				{"a": mockDynamic{value: "ks_111"}},
				{"a": mockDynamic{value: "ks_222"}},
				{"a": mockDynamic{value: "ks_111"}}, // duplicate
				{"a": mockDynamic{value: "ks_333"}},
				{"a": mockDynamic{value: "ks_222"}}, // duplicate
			},
			mappings: []ColumnMapping{
				{ResultColumn: "a", ActualColumn: "key_space_id"},
			},
			expectedResults: []map[string]any{
				{"a": "api_111"},
				{"a": "api_222"},
				{"a": "api_111"},
				{"a": "api_333"},
				{"a": "api_222"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := New(tt.columnConfigs)
			result, err := transformer.TransformWithMappings(ctx, tt.results, tt.mappings)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.expectedResults) {
				t.Fatalf("result length = %d, want %d", len(result), len(tt.expectedResults))
			}

			for i, row := range result {
				expectedRow := tt.expectedResults[i]
				if len(row) != len(expectedRow) {
					t.Errorf("row %d: column count = %d, want %d", i, len(row), len(expectedRow))
				}

				for col, val := range expectedRow {
					gotVal, exists := row[col]
					if !exists {
						t.Errorf("row %d: missing column %q", i, col)
						continue
					}

					// For mockDynamic values, compare the wrapped value
					if expectedMock, ok := val.(mockDynamic); ok {
						if gotMock, ok := gotVal.(mockDynamic); ok {
							if gotMock.value != expectedMock.value {
								t.Errorf("row %d, column %q: got %q, want %q", i, col, gotMock.value, expectedMock.value)
							}
						} else {
							t.Errorf("row %d, column %q: expected mockDynamic, got %T", i, col, gotVal)
						}
					} else {
						// For transformed string values
						if gotVal != val {
							t.Errorf("row %d, column %q: got %v, want %v", i, col, gotVal, val)
						}
					}
				}
			}
		})
	}
}

// mockDynamic simulates ch.Dynamic behavior by wrapping values in braces
type mockDynamic struct {
	value string
}

func (m mockDynamic) String() string {
	return fmt.Sprintf("{%s }", m.value)
}
