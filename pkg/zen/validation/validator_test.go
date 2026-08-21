package validation_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	openapivalidation "github.com/unkeyed/unkey/pkg/openapi/validation"
	zenvalidation "github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestSchemaIsValid(t *testing.T) {
	coreValidator, err := openapivalidation.NewFromBytes(openapi.Spec)
	require.NoError(t, err)
	require.NotNil(t, zenvalidation.New(coreValidator))
}
