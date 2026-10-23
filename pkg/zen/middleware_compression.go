package zen

import (
	"context"
	"net/http"
	"strconv"
	"strings"
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
	if !hasToken(headers.Values("Vary"), "Accept-Encoding") && !hasToken(headers.Values("Vary"), "*") {
		headers.Add("Vary", "Accept-Encoding")
	}
	if len(body) < compressionThreshold || headers.Get("Content-Encoding") != "" ||
		hasToken(headers.Values("Cache-Control"), "no-transform") ||
		hasToken(s.Request().Header.Values("Cache-Control"), "no-transform") {
		return s.writeResponse(status, body)
	}

	switch selectEncoding(s.Request().Header.Values("Accept-Encoding")) {
	case "gzip":
		return writeGzip(s, body)
	case "zstd":
		return writeZstd(s, body)
	default:
		return s.writeResponse(status, body)
	}
}

func writeEncoded(s *Session, body, encoded []byte, encoding string) error {
	if len(encoded) >= len(body) {
		return s.writeResponse(http.StatusOK, body)
	}
	headers := s.ResponseWriter().Header()
	headers.Set("Content-Encoding", encoding)
	headers.Set("Content-Length", strconv.Itoa(len(encoded)))
	return s.writeResponse(http.StatusOK, encoded)
}
