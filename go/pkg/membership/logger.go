package membership

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/unkeyed/unkey/go/pkg/logging"
)

// logger implements io.Writer interface to integrate with memberlist's logging system
// and routes logs to the application's structured logging system.
type logger struct {
	logger logging.Logger
}

// Write implements io.Writer and processes memberlist log messages.
// It parses the log level from the message format [LEVEL] and routes
// the message to the appropriate logging level in the application's
// structured logging system.
//
// Log format expected: [LEVEL] message
// Supported levels: DEBUG, INFO, WARN, ERROR
func (l logger) Write(p []byte) (n int, err error) {

	var level string
	x := bytes.IndexByte(p, '[')
	if x >= 0 {
		y := bytes.IndexByte(p[x:], ']')

		if y >= 0 {
			level = string(p[x+1 : x+y])

		}
	}

	switch level {
	case "DEBUG":
		break
	case "INFO":
		l.logger.Info(context.Background(), string(p), slog.String("pkg", "memberlist"))
	case "WARN":
		l.logger.Warn(context.Background(), string(p), slog.String("pkg", "memberlist"))
	case "ERROR":
		l.logger.Error(context.Background(), string(p), slog.String("pkg", "memberlist"))
	}
	return len(p), nil
}
