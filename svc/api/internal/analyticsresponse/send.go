package analyticsresponse

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

const compressionThreshold = 1024

var gzipEncoders sync.Pool
var zstdEncoders sync.Pool

type gzipEncoder struct {
	writer *gzip.Writer
	buffer bytes.Buffer
}

type zstdEncoder struct {
	writer *zstd.Encoder
	buffer []byte
}

// Send writes a successful JSON response, compressing bodies of at least 1 KiB
// when the client accepts gzip or zstd. Unsupported encodings fall back to JSON.
// The caller must enforce the response size limit before calling Send.
func Send(s *zen.Session, body []byte) error {
	headers := s.ResponseWriter().Header()
	headers.Set("Content-Type", "application/json")
	if !hasToken(headers.Values("Vary"), "Accept-Encoding") && !hasToken(headers.Values("Vary"), "*") {
		headers.Add("Vary", "Accept-Encoding")
	}
	if len(body) < compressionThreshold || headers.Get("Content-Encoding") != "" ||
		hasToken(headers.Values("Cache-Control"), "no-transform") ||
		hasToken(s.Request().Header.Values("Cache-Control"), "no-transform") {
		return s.Send(http.StatusOK, body)
	}

	switch selectEncoding(s.Request().Header.Values("Accept-Encoding")) {
	case "gzip":
		return sendGzip(s, body)
	case "zstd":
		return sendZstd(s, body)
	default:
		return s.Send(http.StatusOK, body)
	}
}

func sendGzip(s *zen.Session, body []byte) error {
	var encoder *gzipEncoder
	if cached := gzipEncoders.Get(); cached != nil {
		encoder = cached.(*gzipEncoder)
	} else {
		writer, err := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		if err != nil {
			return fault.Wrap(err, fault.Internal("create analytics gzip encoder"))
		}
		encoder = &gzipEncoder{writer: writer, buffer: bytes.Buffer{}}
	}
	defer gzipEncoders.Put(encoder)
	encoder.buffer.Reset()
	encoder.writer.Reset(&encoder.buffer)
	if _, err := encoder.writer.Write(body); err != nil {
		return fault.Wrap(err, fault.Internal("compress analytics response"))
	}
	if err := encoder.writer.Close(); err != nil {
		return fault.Wrap(err, fault.Internal("finish analytics gzip response"))
	}
	return sendEncoded(s, body, encoder.buffer.Bytes(), "gzip")
}

func sendZstd(s *zen.Session, body []byte) error {
	var encoder *zstdEncoder
	if cached := zstdEncoders.Get(); cached != nil {
		encoder = cached.(*zstdEncoder)
	} else {
		writer, err := zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.SpeedFastest),
			zstd.WithEncoderConcurrency(1),
		)
		if err != nil {
			return fault.Wrap(err, fault.Internal("create analytics zstd encoder"))
		}
		encoder = &zstdEncoder{writer: writer, buffer: nil}
	}
	defer zstdEncoders.Put(encoder)
	encoder.buffer = encoder.writer.EncodeAll(body, encoder.buffer[:0])
	return sendEncoded(s, body, encoder.buffer, "zstd")
}

func sendEncoded(s *zen.Session, body, encoded []byte, encoding string) error {
	if len(encoded) >= len(body) {
		return s.Send(http.StatusOK, body)
	}
	headers := s.ResponseWriter().Header()
	headers.Set("Content-Encoding", encoding)
	headers.Set("Content-Length", strconv.Itoa(len(encoded)))
	err := s.Send(http.StatusOK, encoded)
	s.SetResponseBody(body)
	return err
}

func selectEncoding(values []string) string {
	qualities := make(map[string]float64, 4)
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			name, parameter, hasParameter := strings.Cut(item, ";")
			name = strings.ToLower(strings.TrimSpace(name))
			if name == "x-gzip" {
				name = "gzip"
			}
			quality := 1.0
			if hasParameter {
				key, weight, ok := strings.Cut(parameter, "=")
				parsed, err := strconv.ParseFloat(strings.TrimSpace(weight), 64)
				if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") || err != nil || !(parsed >= 0 && parsed <= 1) {
					quality = 0
				} else {
					quality = parsed
				}
			}
			qualities[name] = max(qualities[name], quality)
		}
	}

	selected := ""
	bestQuality := qualities["identity"]
	for _, encoding := range []string{"zstd", "gzip"} {
		quality, explicit := qualities[encoding]
		if !explicit {
			quality = qualities["*"]
		}
		if quality > bestQuality {
			selected = encoding
			bestQuality = quality
		}
	}
	return selected
}

func hasToken(values []string, token string) bool {
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(item), token) {
				return true
			}
		}
	}
	return false
}
