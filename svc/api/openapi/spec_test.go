package openapi

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestBundledErrorSchemaNames(t *testing.T) {
	document, err := libopenapi.NewDocument(Spec)
	require.NoError(t, err)
	model, err := document.BuildV3Model()
	require.NoError(t, err)
	schemas := model.Model.Components.Schemas
	require.NotNil(t, schemas.GetOrZero("BaseError"))
	require.Nil(t, schemas.GetOrZero("BaseError__error"))

	for _, name := range []string{
		"ConflictErrorResponse", "ForbiddenErrorResponse", "GoneErrorResponse",
		"InternalServerErrorResponse", "NotFoundErrorResponse",
		"PreconditionFailedErrorResponse", "ServiceUnavailableErrorResponse",
		"TooManyRequestsErrorResponse", "UnauthorizedErrorResponse",
		"UnprocessableEntityErrorResponse",
	} {
		t.Run(name, func(t *testing.T) {
			schema := schemas.GetOrZero(name)
			require.NotNil(t, schema)
			errorSchema := schema.Schema().Properties.GetOrZero("error")
			require.NotNil(t, errorSchema)
			require.Equal(t, "#/components/schemas/BaseError", errorSchema.GetReference())
		})
	}
}

func TestOptionalResponseObjectsRemainOmitted(t *testing.T) {
	var key KeyResponseData
	var verification V2KeysVerifyKeyResponseData
	var credits KeyCreditsData
	for _, tc := range []struct {
		name  string
		value any
		field string
	}{
		{"key identity", key, "identity"},
		{"verification identity", verification, "identity"},
		{"credit refill", credits, "refill"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.value)
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(body, &fields))
			require.NotContains(t, fields, tc.field)
		})
	}
}
