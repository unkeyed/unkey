package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatLocation_Body(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "body"},
		{"/", "body"},
		{"/keyId", "body.keyId"},
		{"/roles", "body.roles"},
		{"/roles/0", "body.roles[0]"},
		{"/roles/0/name", "body.roles[0].name"},
		{"/data/items/1/value", "body.data.items[1].value"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FormatLocation("body", tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatLocation_Params(t *testing.T) {
	tests := []struct {
		prefix   string
		pointer  string
		expected string
	}{
		{"query.param", "", "query.param"},
		{"query.param", "/", "query.param"},
		{"query.param", "/0", "query.param[0]"},
		{"query.param", "/items/0", "query.param.items[0]"},
		{"query.param", "/items/0/name", "query.param.items[0].name"},
		{"body", "/roles/0/name", "body.roles[0].name"},
		{"header.X-Custom", "/nested/0", "header.X-Custom.nested[0]"},
		{"path.userId", "", "path.userId"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"_"+tt.pointer, func(t *testing.T) {
			result := FormatLocation(tt.prefix, tt.pointer)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatLocation_ComplexPaths(t *testing.T) {
	tests := []struct {
		prefix   string
		pointer  string
		expected string
	}{
		{"body", "/deeply/nested/0/array/1/value", "body.deeply.nested[0].array[1].value"},
		{"body", "/0/1/2", "body[0][1][2]"},
		{"query.filter", "/conditions/0/field", "query.filter.conditions[0].field"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatLocation(tt.prefix, tt.pointer)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSuggestFix(t *testing.T) {
	tests := []struct {
		keywordLocation string
		message         string
		expectNil       bool
		expectedFix     string
	}{
		{
			keywordLocation: "/properties/keyId/required",
			message:         "missing required property 'keyId'",
			expectNil:       false,
			expectedFix:     "Add the missing required field to your request",
		},
		{
			keywordLocation: "/properties/name/type",
			message:         "expected string, got number",
			expectNil:       false,
			expectedFix:     "Ensure the field has the correct data type",
		},
		{
			keywordLocation: "/properties/name/minLength",
			message:         "string is too short",
			expectNil:       false,
			expectedFix:     "Provide a longer value",
		},
		{
			keywordLocation: "/properties/name/maxLength",
			message:         "string is too long",
			expectNil:       false,
			expectedFix:     "Provide a shorter value",
		},
		{
			keywordLocation: "/properties/status/enum",
			message:         "value is not one of allowed values",
			expectNil:       false,
			expectedFix:     "Use one of the allowed values",
		},
		{
			keywordLocation: "/properties/count/minimum",
			message:         "value is less than minimum",
			expectNil:       false,
			expectedFix:     "Provide a larger value",
		},
		{
			keywordLocation: "/properties/count/maximum",
			message:         "value is greater than maximum",
			expectNil:       false,
			expectedFix:     "Provide a smaller value",
		},
		{
			keywordLocation: "/properties/items/minItems",
			message:         "array has too few items",
			expectNil:       false,
			expectedFix:     "Add more items to the array",
		},
		{
			keywordLocation: "/properties/items/maxItems",
			message:         "array has too many items",
			expectNil:       false,
			expectedFix:     "Remove some items from the array",
		},
		{
			keywordLocation: "/additionalProperties",
			message:         "additional property 'foo' is not allowed",
			expectNil:       false,
			expectedFix:     "Remove the unknown field from your request",
		},
		{
			keywordLocation: "/properties/email/pattern",
			message:         "does not match pattern",
			expectNil:       false,
			expectedFix:     "Ensure the value matches the required format",
		},
		{
			keywordLocation: "/unknownKeyword",
			message:         "unknown validation error",
			expectNil:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.keywordLocation, func(t *testing.T) {
			fix := suggestFix(tt.keywordLocation, tt.message)
			if tt.expectNil {
				require.Nil(t, fix)
			} else {
				require.NotNil(t, fix)
				require.Equal(t, tt.expectedFix, *fix)
			}
		})
	}
}

func TestIsArrayIndex(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0", true},
		{"1", true},
		{"123", true},
		{"", false},
		{"abc", false},
		{"0abc", false},
		{"abc0", false},
		{"-1", false},
		{"1.5", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isArrayIndex(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
