package bus

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/logger"
)

// logWriter adapts Serf/memberlist's io.Writer log output to the structured
// logger. memberlist prefixes lines with "[ERR]", "[WARN]", "[DEBUG]"; we
// route on the prefix and tag with subsystem so operators can filter.
type logWriter struct {
	subsystem string
}

func newLogWriter(subsystem string) *logWriter {
	return &logWriter{subsystem: subsystem}
}

func (w *logWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}

	switch {
	case strings.Contains(msg, "[ERR]"):
		logger.Error(msg, "subsystem", w.subsystem)
	case strings.Contains(msg, "[WARN]"):
		logger.Warn(msg, "subsystem", w.subsystem)
	default:
		logger.Debug(msg, "subsystem", w.subsystem)
	}

	return len(p), nil
}
