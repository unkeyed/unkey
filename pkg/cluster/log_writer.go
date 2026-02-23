package cluster

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/logger"
)

// logWriter adapts memberlist's io.Writer log output to our structured logger.
type logWriter struct {
	pool string
}

func newLogWriter(pool string) *logWriter {
	return &logWriter{pool: pool}
}

func (w *logWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}

	// memberlist prefixes lines with [DEBUG], [WARN], [ERR], etc.
	switch {
	case strings.Contains(msg, "[ERR]"):
		logger.Error(msg, "pool", w.pool)
	case strings.Contains(msg, "[WARN]"):
		logger.Warn(msg, "pool", w.pool)
	default:
		logger.Debug(msg, "pool", w.pool)
	}

	return len(p), nil
}
