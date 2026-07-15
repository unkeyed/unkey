package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaIsValid(t *testing.T) {
	_, err := New()
	require.NoError(t, err)
}
