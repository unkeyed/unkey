package validation

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDiscriminatorExtraction tests that discriminator info is correctly extracted during compilation
func TestDiscriminatorExtraction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		schema             map[string]any
		expectedProperty   string
		expectedMappingLen int
	}{
		{
			name: "explicit mapping",
			schema: map[string]any{
				"oneOf": []any{
					map[string]any{"$ref": "#/components/schemas/Dog"},
					map[string]any{"$ref": "#/components/schemas/Cat"},
				},
				"discriminator": map[string]any{
					"propertyName": "petType",
					"mapping": map[string]any{
						"dog": "#/components/schemas/Dog",
						"cat": "#/components/schemas/Cat",
					},
				},
			},
			expectedProperty:   "petType",
			expectedMappingLen: 2,
		},
		{
			name: "implicit mapping from refs",
			schema: map[string]any{
				"oneOf": []any{
					map[string]any{"$ref": "#/components/schemas/Dog"},
					map[string]any{"$ref": "#/components/schemas/Cat"},
				},
				"discriminator": map[string]any{
					"propertyName": "type",
				},
			},
			expectedProperty:   "type",
			expectedMappingLen: 2,
		},
		{
			name: "no discriminator",
			schema: map[string]any{
				"oneOf": []any{
					map[string]any{"$ref": "#/components/schemas/Dog"},
					map[string]any{"$ref": "#/components/schemas/Cat"},
				},
			},
			expectedProperty:   "",
			expectedMappingLen: 0,
		},
		{
			name: "discriminator without propertyName",
			schema: map[string]any{
				"oneOf": []any{
					map[string]any{"$ref": "#/components/schemas/Dog"},
				},
				"discriminator": map[string]any{
					"mapping": map[string]any{
						"dog": "#/components/schemas/Dog",
					},
				},
			},
			expectedProperty:   "",
			expectedMappingLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Pass nil resolver for these tests since they don't use $ref at the top level
			result := extractDiscriminator(tt.schema, nil)

			if tt.expectedProperty == "" {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, tt.expectedProperty, result.PropertyName)
				require.Len(t, result.Mapping, tt.expectedMappingLen)
			}
		})
	}
}

// TestDiscriminatorValidation tests oneOf validation with discriminator hints
func TestDiscriminatorValidation(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid dog",
			body:       `{"petType": "dog", "bark": true}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid cat",
			body:       `{"petType": "cat", "meow": true}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid - dog with meow instead of bark",
			body:       `{"petType": "dog", "meow": true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid - cat with bark instead of meow",
			body:       `{"petType": "cat", "bark": true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid - missing discriminator property",
			body:       `{"bark": true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid - unknown discriminator value",
			body:       `{"petType": "bird", "wings": true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid - empty object",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := makeRequest(http.MethodPost, "/test/pet", tt.body, nil)
			resp, ok := validator.Validate(context.Background(), req)

			if tt.wantStatus == http.StatusOK {
				require.True(t, ok, "expected validation to pass")
				require.Nil(t, resp, "expected no error response")
			} else {
				require.False(t, ok, "expected validation to fail")
				require.NotNil(t, resp, "expected error response")
			}
		})
	}
}

// TestDiscriminatorCompiledOperation verifies that discriminator info is stored in CompiledOperation
func TestDiscriminatorCompiledOperation(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)

	// Test with inline schema that has discriminator directly in the request schema
	// (not via $ref which doesn't get the discriminator extracted)
	compiledOp := validator.compiler.GetOperation("testPetInline")
	require.NotNil(t, compiledOp, "expected compiled operation for testPetInline")
	require.NotNil(t, compiledOp.Discriminator, "expected discriminator info to be extracted from inline schema")
	require.Equal(t, "petType", compiledOp.Discriminator.PropertyName)
	require.Len(t, compiledOp.Discriminator.Mapping, 2)
	require.Equal(t, "#/components/schemas/Dog", compiledOp.Discriminator.Mapping["dog"])
	require.Equal(t, "#/components/schemas/Cat", compiledOp.Discriminator.Mapping["cat"])

	// For $ref schemas, the discriminator is inside the referenced schema
	// and won't be extracted from op.RequestSchema (which is just the $ref)
	refOp := validator.compiler.GetOperation("testPet")
	require.NotNil(t, refOp, "expected compiled operation for testPet")
	// Discriminator extraction from $ref schemas is not currently supported
	// (would require resolving refs before extraction)
	require.Nil(t, refOp.Discriminator, "discriminator from $ref schemas is not extracted")
}
