package validation

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNullableTypes tests OpenAPI 3.1 nullable types (type: ["string", "null"])
func TestNullableTypes(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid string value",
			body:    `{"value": "hello"}`,
			isValid: true,
		},
		{
			name:    "valid null value",
			body:    `{"value": null}`,
			isValid: true,
		},
		{
			name:    "invalid number value",
			body:    `{"value": 123}`,
			isValid: false,
		},
		{
			name:    "missing required nullable field",
			body:    `{}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/nullable", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestOneOfValidation tests oneOf schema validation
func TestOneOfValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid string variant",
			body:    `{"type": "string", "stringValue": "hello"}`,
			isValid: true,
		},
		{
			name:    "valid number variant",
			body:    `{"type": "number", "numberValue": 42}`,
			isValid: true,
		},
		{
			name:    "invalid - matches neither",
			body:    `{"type": "boolean", "boolValue": true}`,
			isValid: false,
		},
		{
			name:    "invalid - missing required field",
			body:    `{"type": "string"}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/oneof", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestAnyOfValidation tests anyOf schema validation
func TestAnyOfValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "matches first schema",
			body:    `{"name": "test"}`,
			isValid: true,
		},
		{
			name:    "matches second schema",
			body:    `{"title": "test"}`,
			isValid: true,
		},
		{
			name:    "matches both schemas",
			body:    `{"name": "test", "title": "test"}`,
			isValid: true,
		},
		{
			name:    "empty object still valid (anyOf with no required)",
			body:    `{}`,
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/anyof", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestAllOfValidation tests allOf schema validation
func TestAllOfValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid - all required fields present",
			body:    `{"id": "123", "name": "test"}`,
			isValid: true,
		},
		{
			name:    "invalid - missing id",
			body:    `{"name": "test"}`,
			isValid: false,
		},
		{
			name:    "invalid - missing name",
			body:    `{"id": "123"}`,
			isValid: false,
		},
		{
			name:    "invalid - wrong type for id",
			body:    `{"id": 123, "name": "test"}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/allof", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestArrayValidation tests array schema validation with minItems/maxItems
func TestArrayValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid array",
			body:    `[1, 2, 3]`,
			isValid: true,
		},
		{
			name:    "valid single item",
			body:    `[1]`,
			isValid: true,
		},
		{
			name:    "valid max items",
			body:    `[1, 2, 3, 4, 5, 6, 7, 8, 9, 10]`,
			isValid: true,
		},
		{
			name:    "invalid empty array (minItems: 1)",
			body:    `[]`,
			isValid: false,
		},
		{
			name:    "invalid too many items (maxItems: 10)",
			body:    `[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11]`,
			isValid: false,
		},
		{
			name:    "invalid item type",
			body:    `["a", "b", "c"]`,
			isValid: false,
		},
		{
			name:    "invalid mixed types",
			body:    `[1, "two", 3]`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/array", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestNestedObjectValidation tests deeply nested object validation
func TestNestedObjectValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid nested structure",
			body:    `{"level1": {"level2": {"value": "test"}}}`,
			isValid: true,
		},
		{
			name:    "invalid - missing level1",
			body:    `{}`,
			isValid: false,
		},
		{
			name:    "invalid - missing level2",
			body:    `{"level1": {}}`,
			isValid: false,
		},
		{
			name:    "invalid - missing value",
			body:    `{"level1": {"level2": {}}}`,
			isValid: false,
		},
		{
			name:    "invalid - wrong type at nested level",
			body:    `{"level1": {"level2": {"value": 123}}}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/nested", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestEnumValidation tests enum schema validation
func TestEnumValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid enum value - active",
			body:    `{"status": "active"}`,
			isValid: true,
		},
		{
			name:    "valid enum value - inactive",
			body:    `{"status": "inactive"}`,
			isValid: true,
		},
		{
			name:    "valid enum value - pending",
			body:    `{"status": "pending"}`,
			isValid: true,
		},
		{
			name:    "invalid enum value",
			body:    `{"status": "unknown"}`,
			isValid: false,
		},
		{
			name:    "invalid enum value - wrong case",
			body:    `{"status": "ACTIVE"}`,
			isValid: false,
		},
		{
			name:    "invalid enum type",
			body:    `{"status": 1}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/enum", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestPatternValidation tests regex pattern validation
func TestPatternValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid pattern - abc_123",
			body:    `{"code": "abc_123"}`,
			isValid: true,
		},
		{
			name:    "valid pattern - test_1",
			body:    `{"code": "test_1"}`,
			isValid: true,
		},
		{
			name:    "invalid pattern - missing underscore",
			body:    `{"code": "abc123"}`,
			isValid: false,
		},
		{
			name:    "invalid pattern - uppercase",
			body:    `{"code": "ABC_123"}`,
			isValid: false,
		},
		{
			name:    "invalid pattern - numbers before underscore",
			body:    `{"code": "123_abc"}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/pattern", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestMinMaxValidation tests min/max constraints on strings and numbers
func TestMinMaxValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid - count at minimum",
			body:    `{"count": 0}`,
			isValid: true,
		},
		{
			name:    "valid - count at maximum",
			body:    `{"count": 100}`,
			isValid: true,
		},
		{
			name:    "valid - count in range",
			body:    `{"count": 50}`,
			isValid: true,
		},
		{
			name:    "invalid - count below minimum",
			body:    `{"count": -1}`,
			isValid: false,
		},
		{
			name:    "invalid - count above maximum",
			body:    `{"count": 101}`,
			isValid: false,
		},
		{
			name:    "valid - name at minimum length",
			body:    `{"name": "a"}`,
			isValid: true,
		},
		{
			name:    "invalid - name too short",
			body:    `{"name": ""}`,
			isValid: false,
		},
		{
			name:    "invalid - name too long",
			body:    `{"name": "` + string(make([]byte, 51)) + `"}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/minmax", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestInlineSchemaValidation tests inline schemas with various constraints
func TestInlineSchemaValidation(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid - all fields",
			body:    `{"name": "John", "age": 30, "tags": ["developer"]}`,
			isValid: true,
		},
		{
			name:    "valid - required fields only",
			body:    `{"name": "John", "age": 30}`,
			isValid: true,
		},
		{
			name:    "invalid - missing name",
			body:    `{"age": 30}`,
			isValid: false,
		},
		{
			name:    "invalid - missing age",
			body:    `{"name": "John"}`,
			isValid: false,
		},
		{
			name:    "invalid - age too high",
			body:    `{"name": "John", "age": 200}`,
			isValid: false,
		},
		{
			name:    "invalid - too many tags",
			body:    `{"name": "John", "age": 30, "tags": ["a", "b", "c", "d", "e", "f"]}`,
			isValid: false,
		},
		{
			name:    "invalid - additional property",
			body:    `{"name": "John", "age": 30, "unknown": "field"}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/inline-schema", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestRefWithSiblings tests $ref combined with additional properties (allOf pattern)
// Note: When using allOf with a schema that has additionalProperties: false,
// additional properties will be rejected even if defined in another allOf branch.
// This is correct JSON Schema behavior.
func TestRefWithSiblings(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid - base User fields",
			body:    `{"id": "123", "name": "John"}`,
			isValid: true,
		},
		{
			// This fails because User schema has additionalProperties: false
			// which rejects the additional createdAt field in allOf
			name:    "invalid - with additional createdAt (blocked by additionalProperties: false)",
			body:    `{"id": "123", "name": "John", "createdAt": "2024-01-15T10:30:00Z"}`,
			isValid: false,
		},
		{
			name:    "invalid - missing id from base",
			body:    `{"name": "John"}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/ref-with-siblings", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestNoAuthEndpoint tests endpoints with security: []
func TestNoAuthEndpoint(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name     string
		body     string
		withAuth bool
		isValid  bool
	}{
		{
			name:     "valid without auth",
			body:     `{"data": "test"}`,
			withAuth: false,
			isValid:  true,
		},
		{
			name:     "valid with auth (auth is optional)",
			body:     `{"data": "test"}`,
			withAuth: true,
			isValid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/test/no-auth", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			if tt.withAuth {
				req.Header.Set("Authorization", "Bearer test_token")
			}

			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestUserSchema tests the basic User schema validation
func TestUserSchema(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		body    string
		isValid bool
	}{
		{
			name:    "valid user",
			body:    `{"id": "user_123", "name": "John Doe"}`,
			isValid: true,
		},
		{
			name:    "valid user with email",
			body:    `{"id": "user_123", "name": "John Doe", "email": "john@example.com"}`,
			isValid: true,
		},
		{
			name:    "invalid - missing id",
			body:    `{"name": "John Doe"}`,
			isValid: false,
		},
		{
			name:    "invalid - missing name",
			body:    `{"id": "user_123"}`,
			isValid: false,
		},
		{
			name:    "invalid - additional property",
			body:    `{"id": "user_123", "name": "John", "extra": "field"}`,
			isValid: false,
		},
		{
			name:    "invalid - wrong id type",
			body:    `{"id": 123, "name": "John Doe"}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeRequest(http.MethodPost, "/test/user", tt.body, nil)
			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}
