package openapi

import (
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
