package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatError_NilError(t *testing.T) {
	result := FormatError(nil)
	require.Equal(t, "", result)
}

func TestFormatError_GenericError(t *testing.T) {
	err := fmt.Errorf("something went wrong")
	result := FormatError(err)
	require.Equal(t, "something went wrong", result)
}
