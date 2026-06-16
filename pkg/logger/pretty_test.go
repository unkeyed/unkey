package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func prettyLine(t *testing.T, h slog.Handler, r slog.Record) string {
	t.Helper()
	require.NoError(t, h.Handle(context.Background(), r))
	ph, ok := h.(*prettyHandler)
	require.True(t, ok)
	buf, ok := ph.out.(*bytes.Buffer)
	require.True(t, ok)
	return buf.String()
}

func TestPrettyHandler(t *testing.T) {
	ts := time.Date(2026, 6, 10, 16, 20, 53, 242_000_000, time.UTC)

	t.Run("renders timestamp, level, message and attrs", func(t *testing.T) {
		h := newPrettyHandler(&bytes.Buffer{}, slog.LevelInfo)
		r := slog.NewRecord(ts, slog.LevelInfo, "database connected", 0)
		r.AddAttrs(slog.String("pool", "primary"), slog.Int("conns", 10))

		out := prettyLine(t, h, r)
		require.Contains(t, out, "16:20:53.242")
		require.Contains(t, out, "INFO")
		require.Contains(t, out, "database connected")
		require.Contains(t, out, "pool="+ansiReset+"primary")
		require.Contains(t, out, "conns="+ansiReset+"10")
	})

	t.Run("quotes values with spaces", func(t *testing.T) {
		h := newPrettyHandler(&bytes.Buffer{}, slog.LevelInfo)
		r := slog.NewRecord(ts, slog.LevelInfo, "msg", 0)
		r.AddAttrs(slog.String("slug", "local api"))

		out := prettyLine(t, h, r)
		require.Contains(t, out, `slug=`+ansiReset+`"local api"`)
	})

	t.Run("WithAttrs prepends attrs to every record", func(t *testing.T) {
		h := newPrettyHandler(&bytes.Buffer{}, slog.LevelInfo).
			WithAttrs([]slog.Attr{slog.String("service", "seed")})
		r := slog.NewRecord(ts, slog.LevelInfo, "msg", 0)

		out := prettyLine(t, h, r)
		require.Contains(t, out, "service="+ansiReset+"seed")
	})

	t.Run("WithGroup qualifies keys with dotted path", func(t *testing.T) {
		h := newPrettyHandler(&bytes.Buffer{}, slog.LevelInfo).WithGroup("db")
		r := slog.NewRecord(ts, slog.LevelInfo, "msg", 0)
		r.AddAttrs(slog.String("host", "localhost"))

		out := prettyLine(t, h, r)
		require.Contains(t, out, "db.host="+ansiReset+"localhost")
	})

	t.Run("group attrs flatten to dotted keys", func(t *testing.T) {
		h := newPrettyHandler(&bytes.Buffer{}, slog.LevelInfo)
		r := slog.NewRecord(ts, slog.LevelInfo, "msg", 0)
		r.AddAttrs(slog.Group("http", slog.Int("status", 200)))

		out := prettyLine(t, h, r)
		require.Contains(t, out, "http.status="+ansiReset+"200")
	})

	t.Run("levels render distinct colored tags", func(t *testing.T) {
		require.Contains(t, levelTag(slog.LevelDebug), "DEBUG")
		require.Contains(t, levelTag(slog.LevelInfo), "INFO")
		require.Contains(t, levelTag(slog.LevelWarn), "WARN")
		require.Contains(t, levelTag(slog.LevelError), "ERROR")
	})

	t.Run("Enabled honors minimum level", func(t *testing.T) {
		h := newPrettyHandler(&bytes.Buffer{}, slog.LevelWarn)

		require.False(t, h.Enabled(context.Background(), slog.LevelInfo))
		require.True(t, h.Enabled(context.Background(), slog.LevelWarn))
	})
}
