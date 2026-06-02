package middleware

import (
	"context"
	"errors"
	"strings"
	"syscall"
)

// isClientGone reports whether err indicates that the client closed the
// connection before we finished writing a response.
//
// Checks exported sentinels first (context.Canceled, EPIPE, ECONNRESET)
// and falls back to substring matches for unexported errors emitted by
// net/http/http2 such as "http2: stream closed" and "client disconnected".
func isClientGone(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "client disconnected") ||
		strings.Contains(msg, "stream closed") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer")
}
