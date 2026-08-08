package buildqueue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildConcurrency(t *testing.T) {
	require.Equal(t, int32(1), buildConcurrency(0, false))
	require.Equal(t, int32(1), buildConcurrency(0, true))
	require.Equal(t, int32(7), buildConcurrency(7, true))
}
