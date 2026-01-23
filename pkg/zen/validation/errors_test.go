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
		fieldName       string
		expectNil       bool
		expectedFix     string
	}{
		{
			keywordLocation: "/properties/keyId/required",
			message:         "missing required property 'keyId'",
			fieldName:       "keyId",
			expectNil:       false,
			expectedFix:     "Add the 'keyId' field to your request body",
		},
		{
			keywordLocation: "/properties/keyId/required",
			message:         "missing required property 'keyId'",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Add the missing required field to your request",
		},
		{
			keywordLocation: "/properties/name/type",
			message:         "expected string, got number",
			fieldName:       "name",
			expectNil:       false,
			expectedFix:     "The 'name' field must be a string",
		},
		{
			keywordLocation: "/properties/name/type",
			message:         "got string, want number",
			fieldName:       "name",
			expectNil:       false,
			expectedFix:     "The 'name' field must be a number",
		},
		{
			keywordLocation: "/properties/name/type",
			message:         "type mismatch",
			fieldName:       "name",
			expectNil:       false,
			expectedFix:     "The 'name' field has an incorrect data type",
		},
		{
			keywordLocation: "/properties/name/type",
			message:         "expected string, got number",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Ensure the field has the correct data type",
		},
		{
			keywordLocation: "/properties/name/minLength",
			message:         "string is too short",
			fieldName:       "name",
			expectNil:       false,
			expectedFix:     "The 'name' field value is too short",
		},
		{
			keywordLocation: "/properties/name/minLength",
			message:         "string is too short",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Provide a longer value",
		},
		{
			keywordLocation: "/properties/name/maxLength",
			message:         "string is too long",
			fieldName:       "name",
			expectNil:       false,
			expectedFix:     "The 'name' field value is too long",
		},
		{
			keywordLocation: "/properties/name/maxLength",
			message:         "string is too long",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Provide a shorter value",
		},
		{
			keywordLocation: "/properties/status/enum",
			message:         "value is not one of allowed values",
			fieldName:       "status",
			expectNil:       false,
			expectedFix:     "The 'status' field must be one of the allowed values",
		},
		{
			keywordLocation: "/properties/status/enum",
			message:         "value is not one of allowed values",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Use one of the allowed values",
		},
		{
			keywordLocation: "/properties/count/minimum",
			message:         "value is less than minimum",
			fieldName:       "count",
			expectNil:       false,
			expectedFix:     "The 'count' field value is too small",
		},
		{
			keywordLocation: "/properties/count/minimum",
			message:         "value is less than minimum",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Provide a larger value",
		},
		{
			keywordLocation: "/properties/count/maximum",
			message:         "value is greater than maximum",
			fieldName:       "count",
			expectNil:       false,
			expectedFix:     "The 'count' field value is too large",
		},
		{
			keywordLocation: "/properties/count/maximum",
			message:         "value is greater than maximum",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Provide a smaller value",
		},
		{
			keywordLocation: "/properties/items/minItems",
			message:         "array has too few items",
			fieldName:       "items",
			expectNil:       false,
			expectedFix:     "The 'items' array has too few items",
		},
		{
			keywordLocation: "/properties/items/minItems",
			message:         "array has too few items",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Add more items to the array",
		},
		{
			keywordLocation: "/properties/items/maxItems",
			message:         "array has too many items",
			fieldName:       "items",
			expectNil:       false,
			expectedFix:     "The 'items' array has too many items",
		},
		{
			keywordLocation: "/properties/items/maxItems",
			message:         "array has too many items",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Remove some items from the array",
		},
		{
			keywordLocation: "/additionalProperties",
			message:         "additional property 'foo' is not allowed",
			fieldName:       "foo",
			expectNil:       false,
			expectedFix:     "Remove the 'foo' field - it is not allowed",
		},
		{
			keywordLocation: "/additionalProperties",
			message:         "additional property 'foo' is not allowed",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Remove the unknown field from your request",
		},
		{
			keywordLocation: "/properties/email/pattern",
			message:         "does not match pattern",
			fieldName:       "email",
			expectNil:       false,
			expectedFix:     "The 'email' field does not match the required format",
		},
		{
			keywordLocation: "/properties/email/pattern",
			message:         "does not match pattern",
			fieldName:       "",
			expectNil:       false,
			expectedFix:     "Ensure the value matches the required format",
		},
		{
			keywordLocation: "/unknownKeyword",
			message:         "unknown validation error",
			fieldName:       "",
			expectNil:       true,
		},
	}

	for _, tt := range tests {
		name := tt.keywordLocation
		if tt.fieldName != "" {
			name += "_with_field_" + tt.fieldName
		} else {
			name += "_no_field"
		}
		t.Run(name, func(t *testing.T) {
			fix := suggestFix(tt.keywordLocation, tt.message, tt.fieldName)
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

func TestExtractKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/properties/keyId/type", "type"},
		{"/properties/query/required", "required"},
		{"/additionalProperties", "additionalProperties"},
		{"/properties/items/0/properties/name/type", "type"},
		{"", ""},
		{"/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractKeyword(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFieldFromKeywordLocation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/properties/query/required", "query"},
		{"/properties/keyId/type", "keyId"},
		{"/properties/items/0/properties/name/type", "name"},
		{"/additionalProperties", ""},
		{"/properties/foo/additionalProperties", "foo"},
		{"", ""},
		{"/", ""},
		{"/required", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractFieldFromKeywordLocation(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildLocation(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		instanceLoc string
		keywordLoc  string
		expected    string
	}{
		{
			name:        "required at root",
			prefix:      "body",
			instanceLoc: "/",
			keywordLoc:  "/properties/query/required",
			expected:    "body.query",
		},
		{
			name:        "required at root empty instance",
			prefix:      "body",
			instanceLoc: "",
			keywordLoc:  "/properties/name/required",
			expected:    "body.name",
		},
		{
			name:        "additionalProperties at root",
			prefix:      "body",
			instanceLoc: "/",
			keywordLoc:  "/properties/foo/additionalProperties",
			expected:    "body.foo",
		},
		{
			name:        "type error with instance location",
			prefix:      "body",
			instanceLoc: "/count",
			keywordLoc:  "/properties/count/type",
			expected:    "body.count",
		},
		{
			name:        "nested error",
			prefix:      "body",
			instanceLoc: "/roles/0/name",
			keywordLoc:  "/properties/roles/items/properties/name/type",
			expected:    "body.roles[0].name",
		},
		{
			name:        "no field extraction for non-required at root",
			prefix:      "body",
			instanceLoc: "/",
			keywordLoc:  "/properties/query/type",
			expected:    "body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildLocation(tt.prefix, tt.instanceLoc, tt.keywordLoc)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractExpectedType(t *testing.T) {
	tests := []struct {
		message  string
		expected string
	}{
		{"expected string, got number", "string"},
		{"got string, want number", "number"},
		{"expected boolean, got string", "boolean"},
		{"got number, want string", "string"},
		{"type mismatch", ""},
		{"invalid value", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := extractExpectedType(tt.message)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFieldFromMessage(t *testing.T) {
	tests := []struct {
		keyword  string
		message  string
		expected string
	}{
		{"required", "missing property 'query'", "query"},
		{"required", "missing property 'keyId'", "keyId"},
		{"required", "missing property 'foo_bar'", "foo_bar"},
		{"additionalProperties", "additionalProperties 'foo' not allowed", "foo"},
		{"additionalProperties", "additionalProperties 'unknownField' not allowed", "unknownField"},
		{"required", "some other message", ""},
		{"type", "expected string, got number", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		name := tt.keyword + "_" + tt.message
		if name == "_" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			result := extractFieldFromMessage(tt.keyword, tt.message)
			require.Equal(t, tt.expected, result)
		})
	}
}
