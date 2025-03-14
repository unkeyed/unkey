package membership

import (
	"bytes"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// logger implements io.Writer interface to integrate memberlist's logging system
// with the application's structured logging system.
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
		l.logger.Info(string(p), "pkg", "memberlist")
	case "WARN":
		l.logger.Warn(string(p), "pkg", "memberlist")
	case "ERROR":
		l.logger.Error(string(p), "pkg", "memberlist")
	}
	return len(p), nil
}
