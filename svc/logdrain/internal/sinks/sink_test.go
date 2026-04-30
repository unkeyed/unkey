package sinks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeverityNumber(t *testing.T) {
	t.Parallel()

	// Spot-check the load-bearing ones; documented OTLP scale.
	require.Equal(t, int32(5), SeverityNumber("debug"))
	require.Equal(t, int32(9), SeverityNumber("info"))
	require.Equal(t, int32(9), SeverityNumber("")) // default
	require.Equal(t, int32(13), SeverityNumber("warn"))
	require.Equal(t, int32(13), SeverityNumber("warning"))
	require.Equal(t, int32(17), SeverityNumber("error"))
	require.Equal(t, int32(24), SeverityNumber("fatal"))
}

func TestSourceLabel(t *testing.T) {
	t.Parallel()

	require.Equal(t, "runtime", SourceLabel(RecordRuntime))
	require.Equal(t, "request", SourceLabel(RecordRequest))
}
