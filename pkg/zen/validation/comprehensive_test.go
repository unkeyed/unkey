package validation

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// testOpenAPISpec is a minimal OpenAPI 3.1 spec for testing edge cases
const testOpenAPISpec = `
openapi: "3.1.0"
info:
  title: Test API
  version: "1.0.0"

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
  schemas:
    User:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
          format: email
      additionalProperties: false

    NullableString:
      type:
        - string
        - "null"

    OneOfExample:
      oneOf:
        - type: object
          required: [type, stringValue]
          properties:
            type:
              const: "string"
            stringValue:
              type: string
          additionalProperties: false
        - type: object
          required: [type, numberValue]
          properties:
            type:
              const: "number"
            numberValue:
              type: number
          additionalProperties: false

    AnyOfExample:
      anyOf:
        - type: object
          properties:
            name:
              type: string
        - type: object
          properties:
            title:
              type: string

    AllOfExample:
      allOf:
        - type: object
          properties:
            id:
              type: string
        - type: object
          properties:
            name:
              type: string
      required:
        - id
        - name

    ArrayOfIntegers:
      type: array
      items:
        type: integer
      minItems: 1
      maxItems: 10

    NestedObject:
      type: object
      required:
        - level1
      properties:
        level1:
          type: object
          required:
            - level2
          properties:
            level2:
              type: object
              required:
                - value
              properties:
                value:
                  type: string

    EnumExample:
      type: string
      enum:
        - active
        - inactive
        - pending

    PatternExample:
      type: string
      pattern: "^[a-z]+_[0-9]+$"

    MinMaxExample:
      type: object
      properties:
        count:
          type: integer
          minimum: 0
          maximum: 100
        name:
          type: string
          minLength: 1
          maxLength: 50

security:
  - bearerAuth: []

paths:
  /test/user:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/User"
      responses:
        "200":
          description: OK

  /test/nullable:
    post:
      operationId: testNullable
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                value:
                  $ref: "#/components/schemas/NullableString"
              required:
                - value
      responses:
        "200":
          description: OK

  /test/oneof:
    post:
      operationId: testOneOf
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OneOfExample"
      responses:
        "200":
          description: OK

  /test/anyof:
    post:
      operationId: testAnyOf
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/AnyOfExample"
      responses:
        "200":
          description: OK

  /test/allof:
    post:
      operationId: testAllOf
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/AllOfExample"
      responses:
        "200":
          description: OK

  /test/array:
    post:
      operationId: testArray
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ArrayOfIntegers"
      responses:
        "200":
          description: OK

  /test/nested:
    post:
      operationId: testNested
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/NestedObject"
      responses:
        "200":
          description: OK

  /test/enum:
    post:
      operationId: testEnum
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - status
              properties:
                status:
                  $ref: "#/components/schemas/EnumExample"
      responses:
        "200":
          description: OK

  /test/pattern:
    post:
      operationId: testPattern
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - code
              properties:
                code:
                  $ref: "#/components/schemas/PatternExample"
      responses:
        "200":
          description: OK

  /test/minmax:
    post:
      operationId: testMinMax
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/MinMaxExample"
      responses:
        "200":
          description: OK

  /test/params/{id}:
    get:
      operationId: testParams
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
        - name: limit
          in: query
          required: false
          schema:
            type: integer
            minimum: 1
            maximum: 100
        - name: tags
          in: query
          required: false
          style: form
          explode: true
          schema:
            type: array
            items:
              type: string
        - name: X-Request-ID
          in: header
          required: false
          schema:
            type: string
            format: uuid
      responses:
        "200":
          description: OK

  /test/no-auth:
    post:
      operationId: testNoAuth
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                data:
                  type: string
      responses:
        "200":
          description: OK

  /test/inline-schema:
    post:
      operationId: testInlineSchema
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - name
                - age
              properties:
                name:
                  type: string
                  minLength: 1
                age:
                  type: integer
                  minimum: 0
                  maximum: 150
                tags:
                  type: array
                  items:
                    type: string
                  maxItems: 5
              additionalProperties: false
      responses:
        "200":
          description: OK

  /test/ref-with-siblings:
    post:
      operationId: testRefWithSiblings
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
                - $ref: "#/components/schemas/User"
                - type: object
                  properties:
                    createdAt:
                      type: string
                      format: date-time
      responses:
        "200":
          description: OK
`

// newTestValidator creates a validator from the test spec
func newTestValidator(t *testing.T) *Validator {
	t.Helper()

	parser, err := NewSpecParser([]byte(testOpenAPISpec))
	require.NoError(t, err, "failed to parse test spec")

	compiler, err := NewSchemaCompiler(parser, []byte(testOpenAPISpec))
	require.NoError(t, err, "failed to compile schemas")

	matcher := NewPathMatcher(parser.Operations())

	return &Validator{
		matcher:         matcher,
		compiler:        compiler,
		securitySchemes: parser.SecuritySchemes(),
	}
}

func makeRequest(method, path, body string, headers map[string]string) *http.Request {
	var bodyReader *bytes.Reader
	if body != "" {
		bodyReader = bytes.NewReader([]byte(body))
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req
}

// TestNullableTypes tests OpenAPI 3.1 nullable types (type: ["string", "null"])
func TestNullableTypes(t *testing.T) {
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

// TestPathParameters tests path parameter validation
func TestPathParameters(t *testing.T) {
	v := newTestValidator(t)

	tests := []struct {
		name    string
		path    string
		isValid bool
	}{
		{
			name:    "valid path parameter",
			path:    "/test/params/123",
			isValid: true,
		},
		{
			name:    "valid path parameter with special chars",
			path:    "/test/params/abc-123",
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test_token")

			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestQueryParameters tests query parameter validation
func TestQueryParameters(t *testing.T) {
	v := newTestValidator(t)

	tests := []struct {
		name    string
		path    string
		isValid bool
	}{
		{
			name:    "valid without optional params",
			path:    "/test/params/123",
			isValid: true,
		},
		{
			name:    "valid with limit param",
			path:    "/test/params/123?limit=10",
			isValid: true,
		},
		{
			name:    "valid with multiple tags (explode)",
			path:    "/test/params/123?tags=a&tags=b&tags=c",
			isValid: true,
		},
		{
			name:    "invalid limit below minimum",
			path:    "/test/params/123?limit=0",
			isValid: false,
		},
		{
			name:    "invalid limit above maximum",
			path:    "/test/params/123?limit=101",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test_token")

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

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	v := newTestValidator(t)

	t.Run("unicode in string values", func(t *testing.T) {
		body := `{"id": "用户_123", "name": "日本語テスト"}`
		req := makeRequest(http.MethodPost, "/test/user", body, nil)
		resp, valid := v.Validate(context.Background(), req)
		require.True(t, valid, "unicode should be valid, got errors: %+v", resp)
	})

	t.Run("empty string where required", func(t *testing.T) {
		body := `{"id": "", "name": ""}`
		req := makeRequest(http.MethodPost, "/test/user", body, nil)
		// Empty strings are technically valid unless minLength is specified
		_, valid := v.Validate(context.Background(), req)
		require.True(t, valid, "empty strings should be valid for type: string without minLength")
	})

	t.Run("whitespace-only string", func(t *testing.T) {
		body := `{"id": "   ", "name": "   "}`
		req := makeRequest(http.MethodPost, "/test/user", body, nil)
		_, valid := v.Validate(context.Background(), req)
		require.True(t, valid, "whitespace strings should be valid for type: string without pattern")
	})

	t.Run("very large number", func(t *testing.T) {
		body := `{"count": 99999999999999999999}`
		req := makeRequest(http.MethodPost, "/test/minmax", body, nil)
		_, valid := v.Validate(context.Background(), req)
		require.False(t, valid, "number exceeding max should be invalid")
	})

	t.Run("negative zero", func(t *testing.T) {
		body := `{"count": -0}`
		req := makeRequest(http.MethodPost, "/test/minmax", body, nil)
		resp, valid := v.Validate(context.Background(), req)
		require.True(t, valid, "-0 should equal 0 and be valid, got errors: %+v", resp)
	})

	t.Run("floating point for integer field", func(t *testing.T) {
		body := `[1.5, 2.5, 3.5]`
		req := makeRequest(http.MethodPost, "/test/array", body, nil)
		_, valid := v.Validate(context.Background(), req)
		require.False(t, valid, "floats should be invalid for integer array")
	})

	t.Run("null for non-nullable field", func(t *testing.T) {
		body := `{"id": null, "name": "test"}`
		req := makeRequest(http.MethodPost, "/test/user", body, nil)
		_, valid := v.Validate(context.Background(), req)
		require.False(t, valid, "null should be invalid for non-nullable field")
	})
}

// TestCompilerCleanForJSONSchema tests the cleanForJSONSchema function
func TestCompilerCleanForJSONSchema(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "removes nullable keyword",
			input: map[string]any{
				"type":     "string",
				"nullable": true,
			},
			expected: map[string]any{
				"type": "string",
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
			name: "recursively cleans nested objects",
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
						"type": "string",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
