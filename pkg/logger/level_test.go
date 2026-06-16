package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtLevel(t *testing.T) {
	var buf bytes.Buffer
	//nolint:exhaustruct // only Level matters here
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := AtLevel(base, slog.LevelWarn)
	log := slog.New(h)

	log.Info("info-noise")
	log.Warn("warn-keep")
	log.Error("error-keep")

	out := buf.String()
	require.NotContains(t, out, "info-noise", "INFO below the gate must be dropped")
	require.Contains(t, out, "warn-keep")
	require.Contains(t, out, "error-keep")

	require.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	require.True(t, h.Enabled(context.Background(), slog.LevelWarn))
}
