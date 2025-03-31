package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaIsValid(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	valid, errors := v.validator.ValidateDocument()
	require.True(t, valid)
	require.Len(t, errors, 0)
}
