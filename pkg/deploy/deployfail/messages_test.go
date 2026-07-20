package deployfail

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeViolations(t *testing.T) {
	t.Run("deployable settings have no violations", func(t *testing.T) {
		require.Empty(t, RuntimeViolations(8080, 250, 256))
		require.Empty(t, RuntimeViolations(1, 250, 256), "minimums are inclusive")
		require.Empty(t, RuntimeViolations(65535, 250, 256))
	})

	t.Run("port bounds", func(t *testing.T) {
		low := RuntimeViolations(0, 250, 256)
		require.Len(t, low, 1)
		require.Equal(t, MsgPortTooLow, low[0].Message)
		require.Equal(t, int32(0), low[0].Actual)

		high := RuntimeViolations(65536, 250, 256)
		require.Len(t, high, 1)
		require.Equal(t, MsgPortTooHigh, high[0].Message)
	})

	t.Run("cpu and memory must meet minimums", func(t *testing.T) {
		require.Empty(t, RuntimeViolations(8080, 250, 256), "at the minimum is allowed")

		cpu := RuntimeViolations(8080, 249, 256)
		require.Len(t, cpu, 1)
		require.Equal(t, MsgCPUTooLow, cpu[0].Message)
		require.Equal(t, int32(249), cpu[0].Actual)

		mem := RuntimeViolations(8080, 250, 255)
		require.Len(t, mem, 1)
		require.Equal(t, MsgMemoryTooLow, mem[0].Message)
		require.Equal(t, int32(255), mem[0].Actual)
	})

	t.Run("reports every failing field at once", func(t *testing.T) {
		v := RuntimeViolations(0, 0, 0)
		require.Len(t, v, 3, "port, cpu, memory")
		messages := []string{v[0].Message, v[1].Message, v[2].Message}
		require.Equal(t, []string{MsgPortTooLow, MsgCPUTooLow, MsgMemoryTooLow}, messages)
	})
}
