package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOpenAPISpec = `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /api/test:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - id
                - name
              properties:
                id:
                  type: string
                  description: ID field that is required
                name:
                  type: string
                  description: Name field that is required
                description:
                  type: string
                  nullable: true
                  description: Optional description that can be null
                metadata:
                  type: object
                  nullable: true
                  properties:
                    tags:
                      type: array
                      items:
                        type: string
                    count:
                      type: integer
                      nullable: true
                settings:
                  type: object
                  properties:
                    enabled:
                      type: boolean
                    config:
                      type: object
                      properties:
                        timeout:
                          type: integer
                        retryCount:
                          type: integer
                          nullable: true
                        maxAttempts:
                          type: integer
                          nullable: true
      responses:
        '200':
          description: OK
`

func TestValidatorWithRealSpec(t *testing.T) {
	// Create a validator with our test spec
	document, err := libopenapi.NewDocument([]byte(testOpenAPISpec))
	require.NoError(t, err)

	v, validationErrors := validator.NewValidator(document)
	require.Empty(t, validationErrors)

	testValidator := &Validator{
		validator: v,
	}

	tests := []struct {
		name        string
		body        map[string]interface{}
		shouldPass  bool
		description string
	}{
		{
			name: "valid_request_with_all_fields",
			body: map[string]interface{}{
				"id":          "123",
				"name":        "Test Name",
				"description": "Test Description",
				"metadata": map[string]interface{}{
					"tags":  []string{"tag1", "tag2"},
					"count": 5,
				},
				"settings": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"timeout":     30,
						"retryCount":  3,
						"maxAttempts": 10,
					},
				},
			},
			shouldPass:  true,
			description: "All fields provided with valid values",
		},
		{
			name: "valid_request_with_null_nullable_fields",
			body: map[string]interface{}{
				"id":          "123",
				"name":        "Test Name",
				"description": nil, // nullable field set to null
				"metadata":    nil, // nullable object set to null
				"settings": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"timeout":     30,
						"retryCount":  nullable.NewNullNullable[int](), // nested nullable field set to null
						"maxAttempts": nullable.NewNullNullable[int](), // nested nullable field set to null
					},
				},
			},
			shouldPass:  true,
			description: "Nullable fields can be null",
		},
		{
			name: "valid_request_omitting_nullable_fields",
			body: map[string]interface{}{
				"id":   "123",
				"name": "Test Name",
				// description omitted (nullable)
				// metadata omitted (nullable)
				"settings": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"timeout": 30,
						// retryCount omitted (nullable)
						// maxAttempts omitted (nullable)
					},
				},
			},
			shouldPass:  true,
			description: "Nullable fields can be omitted",
		},
		{
			name: "invalid_missing_required_field",
			body: map[string]interface{}{
				"id": "123",
				// name is missing (required)
			},
			shouldPass:  false,
			description: "Missing required field should fail",
		},
		{
			name: "invalid_null_for_required_field",
			body: map[string]interface{}{
				"id":   "123",
				"name": nil, // required field cannot be null
			},
			shouldPass:  false,
			description: "Required field cannot be null",
		},
		{
			name: "valid_nested_nullable_in_metadata",
			body: map[string]interface{}{
				"id":   "123",
				"name": "Test Name",
				"metadata": map[string]interface{}{
					"tags":  []string{"tag1"},
					"count": nil, // nested nullable field
				},
			},
			shouldPass:  true,
			description: "Nested nullable field in metadata can be null",
		},
		{
			name: "valid_deeply_nested_nullable",
			body: map[string]interface{}{
				"id":   "123",
				"name": "Test Name",
				"settings": map[string]interface{}{
					"enabled": false,
					"config": map[string]interface{}{
						"timeout":     60,
						"retryCount":  nil, // deeply nested nullable
						"maxAttempts": nil, // deeply nested nullable
					},
				},
			},
			shouldPass:  true,
			description: "Deeply nested nullable fields can be null",
		},
		{
			name: "invalid_null_for_non_nullable_nested",
			body: map[string]interface{}{
				"id":   "123",
				"name": "Test Name",
				"settings": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"timeout": nil, // non-nullable field cannot be null
					},
				},
			},
			shouldPass:  false,
			description: "Non-nullable nested field cannot be null",
		},
		{
			name: "valid_mixed_nullable_and_non_nullable",
			body: map[string]interface{}{
				"id":          "456",
				"name":        "Another Test",
				"description": nil, // nullable
				"metadata": map[string]interface{}{
					"tags":  []string{}, // empty array is valid
					"count": nil,        // nullable
				},
				"settings": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"timeout":     0,   // zero value is valid for non-nullable
						"retryCount":  nil, // nullable
						"maxAttempts": 5,   // valid value
					},
				},
			},
			shouldPass:  true,
			description: "Mix of nullable and non-nullable fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Validate the request
			ctx := context.Background()
			_, isValid := testValidator.Validate(ctx, req)

			if tt.shouldPass {
				assert.True(t, isValid, "Expected validation to pass for %s: %s", tt.name, tt.description)
			} else {
				assert.False(t, isValid, "Expected validation to fail for %s: %s", tt.name, tt.description)
			}
		})
	}
}

func TestValidatorComplexNestedNullable(t *testing.T) {
	const complexSpec = `
openapi: 3.0.0
info:
  title: Complex Nested API
  version: 1.0.0
paths:
  /api/complex:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - root
              properties:
                root:
                  type: object
                  required:
                    - level1
                  properties:
                    level1:
                      type: object
                      required:
                        - required_field
                      properties:
                        required_field:
                          type: string
                        level2:
                          type: object
                          nullable: true
                          properties:
                            level3:
                              type: object
                              nullable: true
                              properties:
                                level4:
                                  type: object
                                  nullable: true
                                  properties:
                                    final_value:
                                      type: string
                                      nullable: true
      responses:
        '200':
          description: OK
`

	document, err := libopenapi.NewDocument([]byte(complexSpec))
	require.NoError(t, err)

	v, validationErrors := validator.NewValidator(document)
	require.Empty(t, validationErrors)

	testValidator := &Validator{
		validator: v,
	}

	tests := []struct {
		name       string
		body       map[string]interface{}
		shouldPass bool
	}{
		{
			name: "all_levels_null",
			body: map[string]interface{}{
				"root": map[string]interface{}{
					"level1": map[string]interface{}{
						"required_field": "test",
						"level2":         nil,
					},
				},
			},
			shouldPass: true,
		},
		{
			name: "nested_nulls_at_different_levels",
			body: map[string]interface{}{
				"root": map[string]interface{}{
					"level1": map[string]interface{}{
						"required_field": "test",
						"level2": map[string]interface{}{
							"level3": nil,
						},
					},
				},
			},
			shouldPass: true,
		},
		{
			name: "deeply_nested_null_value",
			body: map[string]interface{}{
				"root": map[string]interface{}{
					"level1": map[string]interface{}{
						"required_field": "test",
						"level2": map[string]interface{}{
							"level3": map[string]interface{}{
								"level4": map[string]interface{}{
									"final_value": nil,
								},
							},
						},
					},
				},
			},
			shouldPass: true,
		},
		{
			name: "all_values_present",
			body: map[string]interface{}{
				"root": map[string]interface{}{
					"level1": map[string]interface{}{
						"required_field": "test",
						"level2": map[string]interface{}{
							"level3": map[string]interface{}{
								"level4": map[string]interface{}{
									"final_value": "end",
								},
							},
						},
					},
				},
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/complex", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.Background()
			_, isValid := testValidator.Validate(ctx, req)

			assert.Equal(t, tt.shouldPass, isValid, "Expected validation result to be %v for %s", tt.shouldPass, tt.name)
		})
	}
}

func TestValidatorMixedNullableAndOtherErrors(t *testing.T) {
	const mixedErrorSpec = `
openapi: 3.0.0
info:
  title: Mixed Error Test API
  version: 1.0.0
paths:
  /api/mixed:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - id
                - email
                - age
              properties:
                id:
                  type: string
                  minLength: 5
                  maxLength: 10
                email:
                  type: string
                  format: email
                age:
                  type: integer
                  minimum: 18
                  maximum: 100
                nickname:
                  type: string
                  nullable: true
                  minLength: 3
                preferences:
                  type: object
                  nullable: true
                  properties:
                    theme:
                      type: string
                      enum: ["light", "dark"]
                    notifications:
                      type: boolean
                score:
                  type: number
                  nullable: true
                  minimum: 0
                  maximum: 100
      responses:
        '200':
          description: OK
`

	document, err := libopenapi.NewDocument([]byte(mixedErrorSpec))
	require.NoError(t, err)

	v, validationErrors := validator.NewValidator(document)
	require.Empty(t, validationErrors)

	testValidator := &Validator{
		validator: v,
	}

	tests := []struct {
		name                  string
		body                  map[string]interface{}
		shouldPass            bool
		expectedErrorContains string
		shouldNotContain      string
	}{
		{
			name: "multiple_errors_with_nullable_fields_null",
			body: map[string]interface{}{
				"id":          "abc",          // too short (min 5)
				"email":       "not-an-email", // invalid format
				"age":         150,            // exceeds maximum
				"nickname":    nil,            // nullable - this error should be ignored
				"preferences": nil,            // nullable - this error should be ignored
				"score":       nil,            // nullable - this error should be ignored
			},
			shouldPass:            false,
			expectedErrorContains: "age", // Should report one of the validation errors
			shouldNotContain:      "nullable",
		},
		{
			name: "invalid_enum_with_nullable_errors",
			body: map[string]interface{}{
				"id":       "valid123",
				"email":    "test@example.com",
				"age":      25,
				"nickname": nil, // nullable - should be ignored
				"preferences": map[string]interface{}{
					"theme":         "invalid-theme", // invalid enum value
					"notifications": true,
				},
				"score": nil, // nullable - should be ignored
			},
			shouldPass:            false,
			expectedErrorContains: "theme", // Should report enum error
			shouldNotContain:      "nullable",
		},
		{
			name: "type_mismatch_with_nullable_errors",
			body: map[string]interface{}{
				"id":       "valid123",
				"email":    "test@example.com",
				"age":      "not-a-number", // type mismatch
				"nickname": nil,            // nullable - should be ignored
				"score":    nil,            // nullable - should be ignored
			},
			shouldPass:            false,
			expectedErrorContains: "age", // Should report type error
			shouldNotContain:      "nullable",
		},
		{
			name: "missing_required_with_nullable_null",
			body: map[string]interface{}{
				// "id" is missing - required field
				"email":       "test@example.com",
				"age":         30,
				"nickname":    nil, // nullable - should be ignored
				"preferences": nil, // nullable - should be ignored
			},
			shouldPass:            false,
			expectedErrorContains: "id", // Should report missing required field
			shouldNotContain:      "nullable",
		},
		{
			name: "all_valid_with_nulls",
			body: map[string]interface{}{
				"id":          "valid123",
				"email":       "test@example.com",
				"age":         30,
				"nickname":    nil, // nullable - valid
				"preferences": nil, // nullable - valid
				"score":       nil, // nullable - valid
			},
			shouldPass: true,
		},
		{
			name: "multiple_validation_errors_no_nullable",
			body: map[string]interface{}{
				"id":    "ab",      // too short
				"email": "invalid", // bad format
				"age":   200,       // too high
				// No nullable fields with null values
			},
			shouldPass:            false,
			expectedErrorContains: "age", // Should report one of the errors
			shouldNotContain:      "nullable",
		},
		{
			name: "constraint_violation_on_nullable_field_when_not_null",
			body: map[string]interface{}{
				"id":       "valid123",
				"email":    "test@example.com",
				"age":      30,
				"nickname": "ab", // too short (min 3) - even though nullable, when provided must be valid
				"score":    150,  // exceeds maximum
			},
			shouldPass:            false,
			expectedErrorContains: "nickname", // Should report constraint violation
			shouldNotContain:      "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/mixed", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.Background()
			response, isValid := testValidator.Validate(ctx, req)

			assert.Equal(t, tt.shouldPass, isValid, "Expected validation result to be %v for %s", tt.shouldPass, tt.name)

			if !tt.shouldPass {
				// Check that the error message contains what we expect
				if tt.expectedErrorContains != "" {
					found := false
					for _, err := range response.Error.Errors {
						if strings.Contains(err.Message, tt.expectedErrorContains) ||
							strings.Contains(err.Location, tt.expectedErrorContains) {
							found = true
							break
						}
					}
					assert.True(t, found,
						"Expected error to contain '%s' in Message or Location fields but not found in errors: %+v",
						tt.expectedErrorContains, response.Error.Errors)
				}

				// Check that nullable-related errors are filtered out
				if tt.shouldNotContain != "" {
					found := false
					for _, err := range response.Error.Errors {
						if strings.Contains(err.Message, tt.shouldNotContain) ||
							strings.Contains(err.Location, tt.shouldNotContain) {
							found = true
							break
						}
					}
					assert.False(t, found,
						"Error should not contain '%s' in Message or Location fields but found in errors: %+v",
						tt.shouldNotContain, response.Error.Errors)
				}

				// Ensure we're not getting nullable errors when other errors exist
				assert.NotEmpty(t, response.Error.Detail, "Should have error details")
			}
		})
	}
}

func TestValidatorArrayWithNullableItems(t *testing.T) {
	const arraySpec = `
openapi: 3.0.0
info:
  title: Array Test API
  version: 1.0.0
paths:
  /api/arrays:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - tags
                - scores
              properties:
                tags:
                  type: array
                  items:
                    type: string
                    nullable: true
                    minLength: 2
                scores:
                  type: array
                  items:
                    type: number
                    nullable: true
                    minimum: 0
                    maximum: 100
                users:
                  type: array
                  nullable: true
                  items:
                    type: object
                    nullable: true
                    properties:
                      name:
                        type: string
                        nullable: true
                      age:
                        type: integer
                        nullable: true
                        minimum: 0
                mixedArray:
                  type: array
                  items:
                    oneOf:
                      - type: [string, number, null]
      responses:
        '200':
          description: OK
`

	document, err := libopenapi.NewDocument([]byte(arraySpec))
	require.NoError(t, err)

	v, validationErrors := validator.NewValidator(document)
	require.Empty(t, validationErrors)

	testValidator := &Validator{
		validator: v,
	}

	tests := []struct {
		name                  string
		body                  map[string]interface{}
		shouldPass            bool
		expectedErrorContains string
		shouldNotContain      string
	}{
		{
			name: "array_with_null_items_should_pass",
			body: map[string]interface{}{
				"tags":   []interface{}{"valid", nil, "ok", nil}, // nullable items
				"scores": []interface{}{50.5, nil, 75.0, nil},    // nullable numbers
			},
			shouldPass: true,
		},
		{
			name: "nested_object_array_with_nulls",
			body: map[string]interface{}{
				"tags":   []interface{}{"valid"},
				"scores": []interface{}{50.5},
				"users": []interface{}{
					map[string]interface{}{
						"name": "Alice",
						"age":  nil, // nullable field in object
					},
					nil, // entire object is nullable
					map[string]interface{}{
						"name": nil, // nullable field
						"age":  25,
					},
				},
			},
			shouldPass: true,
		},
		{
			name: "array_with_invalid_non_null_items",
			body: map[string]interface{}{
				"tags":   []interface{}{"a", nil, "valid", nil}, // "a" is too short (min 2)
				"scores": []interface{}{50.5, nil, 75.0},
			},
			shouldPass:            false,
			expectedErrorContains: "tags", // Should report the minLength violation
			shouldNotContain:      "null",
		},
		{
			name: "array_with_invalid_score",
			body: map[string]interface{}{
				"tags":   []interface{}{"valid", nil},
				"scores": []interface{}{50.5, nil, 150}, // 150 exceeds maximum
			},
			shouldPass:            false,
			expectedErrorContains: "scores", // Should report the maximum violation
			shouldNotContain:      "null",
		},
		{
			name: "null_entire_nullable_array",
			body: map[string]interface{}{
				"tags":   []interface{}{"valid"},
				"scores": []interface{}{50.5},
				"users":  nil, // entire array is nullable
			},
			shouldPass: true,
		},
		{
			name: "mixed_array_with_nulls",
			body: map[string]interface{}{
				"tags":       []interface{}{"valid"},
				"scores":     []interface{}{50.5},
				"mixedArray": []interface{}{"text", nil, 42, nil}, // oneOf with nullable types
			},
			shouldPass: true,
		},
		{
			name: "empty_arrays_are_valid",
			body: map[string]interface{}{
				"tags":   []interface{}{},
				"scores": []interface{}{},
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/arrays", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.Background()
			response, isValid := testValidator.Validate(ctx, req)

			assert.Equal(t, tt.shouldPass, isValid, "Expected validation result to be %v for %s", tt.shouldPass, tt.name)

			if !tt.shouldPass {
				// Check that the error message contains what we expect
				if tt.expectedErrorContains != "" {
					found := false
					for _, err := range response.Error.Errors {
						if strings.Contains(err.Message, tt.expectedErrorContains) ||
							strings.Contains(err.Location, tt.expectedErrorContains) {
							found = true
							break
						}
					}
					assert.True(t, found,
						"Expected error to contain '%s' in Message or Location fields but not found in errors: %+v",
						tt.expectedErrorContains, response.Error.Errors)
				}

				// Check that nullable-related errors are filtered out
				if tt.shouldNotContain != "" {
					found := false
					for _, err := range response.Error.Errors {
						if strings.Contains(err.Message, tt.shouldNotContain) ||
							strings.Contains(err.Location, tt.shouldNotContain) {
							found = true
							break
						}
					}
					assert.False(t, found,
						"Error should not contain '%s' in Message or Location fields but found in errors: %+v",
						tt.shouldNotContain, response.Error.Errors)
				}

				// Ensure we're not getting nullable errors when other errors exist
				assert.NotEmpty(t, response.Error.Detail, "Should have error details")
			}
		})
	}
}
