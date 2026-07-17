package deployfail

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeViolations(t *testing.T) {
	t.Run("deployable settings have no violations", func(t *testing.T) {
		require.Empty(t, RuntimeViolations(8080, 250, 256))
		require.Empty(t, RuntimeViolations(1, 1, 1))
		require.Empty(t, RuntimeViolations(65535, 1, 1))
	})

	t.Run("port bounds", func(t *testing.T) {
		low := RuntimeViolations(0, 1, 1)
		require.Len(t, low, 1)
		require.Equal(t, MsgPortTooLow, low[0].Message)
		require.Equal(t, int32(0), low[0].Actual)

		high := RuntimeViolations(65536, 1, 1)
		require.Len(t, high, 1)
		require.Equal(t, MsgPortTooHigh, high[0].Message)
	})

	t.Run("cpu and memory must be positive", func(t *testing.T) {
		require.Equal(t, MsgCPUTooLow, RuntimeViolations(8080, 0, 1)[0].Message)
		require.Equal(t, MsgMemoryTooLow, RuntimeViolations(8080, 1, 0)[0].Message)
	})

	t.Run("reports every failing field at once", func(t *testing.T) {
		v := RuntimeViolations(0, 0, 0)
		require.Len(t, v, 3, "port, cpu, memory")
		messages := []string{v[0].Message, v[1].Message, v[2].Message}
		require.Equal(t, []string{MsgPortTooLow, MsgCPUTooLow, MsgMemoryTooLow}, messages)
	})
}
