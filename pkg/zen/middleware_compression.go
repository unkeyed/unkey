package zen

import (
	"context"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/zen/compression"
	"golang.org/x/net/http/httpguts"
)

const compressionThreshold = 1024

// WithCompression negotiates gzip or zstd for 200 JSON responses of at least
// 1 KiB sent through Session. Direct ResponseWriter writes are unchanged.
func WithCompression() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			previous := s.sendResponse
			s.sendResponse = writeCompressed
			defer func() { s.sendResponse = previous }()
			return next(ctx, s)
		}
	}
}

func writeCompressed(s *Session, status int, body []byte) error {
	headers := s.ResponseWriter().Header()
	contentType, _, _ := strings.Cut(headers.Get("Content-Type"), ";")
	if status != http.StatusOK || !strings.EqualFold(strings.TrimSpace(contentType), "application/json") {
		return s.writeResponse(status, body)
	}
	if !httpguts.HeaderValuesContainsToken(headers.Values("Vary"), "Accept-Encoding") && !httpguts.HeaderValuesContainsToken(headers.Values("Vary"), "*") {
		headers.Add("Vary", "Accept-Encoding")
	}
	if len(body) < compressionThreshold || headers.Get("Content-Encoding") != "" ||
		httpguts.HeaderValuesContainsToken(headers.Values("Cache-Control"), "no-transform") ||
		httpguts.HeaderValuesContainsToken(s.Request().Header.Values("Cache-Control"), "no-transform") {
		return s.writeResponse(status, body)
	}

	return compression.WriteResponse(s.ResponseWriter(), body, s.Request().Header.Values("Accept-Encoding"))
}
