package zen_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestCompressionNegotiatesEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		accept   []string
		encoding string
	}{
		{name: "absent"},
		{name: "empty", accept: []string{""}},
		{name: "unsupported", accept: []string{"br"}},
		{name: "gzip", accept: []string{"gzip"}, encoding: "gzip"},
		{name: "legacy gzip", accept: []string{"x-gzip"}, encoding: "gzip"},
		{name: "zstd", accept: []string{"zstd"}, encoding: "zstd"},
		{name: "prefer zstd on tie", accept: []string{"gzip, zstd"}, encoding: "zstd"},
		{name: "prefer higher quality", accept: []string{"zstd;q=0.2, gzip;q=0.8"}, encoding: "gzip"},
		{name: "separate header lines", accept: []string{"gzip;q=0.8", "zstd;q=0.2"}, encoding: "gzip"},
		{name: "case and whitespace", accept: []string{" GZIP ; Q=0.3, ZSTD ; q=0.7 "}, encoding: "zstd"},
		{name: "exclude zstd", accept: []string{"zstd;q=0, gzip"}, encoding: "gzip"},
		{name: "exclude both", accept: []string{"gzip;q=0, zstd;q=0"}},
		{name: "wildcard", accept: []string{"*;q=0.5"}, encoding: "zstd"},
		{name: "explicit exclusion beats wildcard", accept: []string{"zstd;q=0, *;q=1"}, encoding: "gzip"},
		{name: "explicit quality beats wildcard", accept: []string{"gzip;q=0.9, zstd;q=0.2, *;q=1"}, encoding: "gzip"},
		{name: "wildcard exclusion", accept: []string{"*;q=0"}},
		{name: "explicit acceptance beats wildcard exclusion", accept: []string{"*;q=0, gzip;q=0.3"}, encoding: "gzip"},
		{name: "prefer identity", accept: []string{"identity;q=1, gzip;q=0.5, zstd;q=0.2"}},
		{name: "prefer compression over identity", accept: []string{"identity;q=0.1, zstd;q=0.5"}, encoding: "zstd"},
		{name: "malformed quality", accept: []string{"zstd;q=invalid, gzip"}, encoding: "gzip"},
		{name: "nonfinite quality", accept: []string{"zstd;q=NaN, gzip"}, encoding: "gzip"},
		{name: "out of range quality", accept: []string{"zstd;q=1.1, gzip"}, encoding: "gzip"},
		{name: "negative quality", accept: []string{"zstd;q=-0.1, gzip"}, encoding: "gzip"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(struct {
				Name string `json:"name"`
				Data string `json:"data"`
			}{Name: tt.name, Data: strings.Repeat(tt.name, 2048)})
			require.NoError(t, err)
			s, rec := newSession(t, http.Header{"Accept-Encoding": tt.accept})
			rec.Header().Set("Vary", "Origin")
			rec.Header().Set("Content-Length", strconv.Itoa(len(body)))
			require.NoError(t, sendCompressedJSON(t, s, body))
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			require.Equal(t, tt.encoding, rec.Header().Get("Content-Encoding"))
			require.Equal(t, []string{"Origin", "Accept-Encoding"}, rec.Header().Values("Vary"))
			require.Equal(t, strconv.Itoa(rec.Body.Len()), rec.Header().Get("Content-Length"))
			require.Equal(t, body, decode(t, rec))
		})
	}
}

func TestCompressionThreshold(t *testing.T) {
	t.Parallel()
	for _, size := range []int{64, 1023, 1024, 1025} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			body := jsonBody(t, size)
			s, rec := newSession(t, http.Header{"Accept-Encoding": {"zstd, gzip"}})
			require.NoError(t, sendCompressedJSON(t, s, body))
			if size < 1024 {
				require.Empty(t, rec.Header().Get("Content-Encoding"))
			} else {
				require.Equal(t, "zstd", rec.Header().Get("Content-Encoding"))
			}
			require.Equal(t, body, decode(t, rec))
		})
	}
}

func TestCompressionPreservesEncodingAndNoTransform(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		request  http.Header
		response http.Header
	}{
		{name: "already encoded", response: http.Header{"Content-Encoding": {"br"}}},
		{name: "response no-transform", response: http.Header{"Cache-Control": {"private, no-transform"}}},
		{name: "request no-transform", request: http.Header{"Cache-Control": {"No-Transform"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := jsonBody(t, 4096)
			headers := http.Header{"Accept-Encoding": {"zstd, gzip"}}
			for name, values := range tt.request {
				headers[name] = values
			}
			s, rec := newSession(t, headers)
			for name, values := range tt.response {
				rec.Header()[name] = values
			}
			require.NoError(t, sendCompressedJSON(t, s, body))
			require.Equal(t, tt.response.Get("Content-Encoding"), rec.Header().Get("Content-Encoding"))
			require.Equal(t, body, rec.Body.Bytes())
		})
	}
}

func TestCompressionPreservesExistingVary(t *testing.T) {
	t.Parallel()
	for _, vary := range []string{"Origin, accept-encoding", "*"} {
		s, rec := newSession(t, http.Header{"Accept-Encoding": {"gzip"}})
		rec.Header().Set("Vary", vary)
		require.NoError(t, sendCompressedJSON(t, s, jsonBody(t, 2048)))
		require.Equal(t, []string{vary}, rec.Header().Values("Vary"))
	}
}

func TestCompressionOnlyAppliesToOptedInJSONResponses(t *testing.T) {
	t.Parallel()
	body := jsonBody(t, 4096)
	for _, tt := range []struct {
		name     string
		handle   func(*zen.Session) error
		status   int
		encoding string
	}{
		{
			name:     "session JSON",
			handle:   func(s *zen.Session) error { return s.JSON(http.StatusOK, json.RawMessage(body)) },
			status:   http.StatusOK,
			encoding: "zstd",
		},
		{
			name:     "error JSON",
			handle:   func(s *zen.Session) error { return s.JSON(http.StatusBadRequest, json.RawMessage(body)) },
			status:   http.StatusBadRequest,
			encoding: "",
		},
		{
			name:     "plain text",
			handle:   func(s *zen.Session) error { return s.Plain(http.StatusOK, body) },
			status:   http.StatusOK,
			encoding: "",
		},
		{
			name: "direct writer",
			handle: func(s *zen.Session) error {
				s.AddHeader("Content-Type", "application/json")
				_, err := s.ResponseWriter().Write(body)
				return err
			},
			status:   http.StatusOK,
			encoding: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, rec := newSession(t, http.Header{"Accept-Encoding": {"zstd"}})
			err := zen.WithCompression()(func(_ context.Context, s *zen.Session) error {
				return tt.handle(s)
			})(t.Context(), s)
			require.NoError(t, err)
			require.Equal(t, tt.status, rec.Code)
			require.Equal(t, tt.status, s.StatusCode())
			require.Equal(t, tt.encoding, rec.Header().Get("Content-Encoding"))
			require.Equal(t, body, decode(t, rec))

			req := httptest.NewRequest(http.MethodPost, "/without-compression", nil)
			req.Header.Set("Accept-Encoding", "zstd")
			rec = httptest.NewRecorder()
			require.NoError(t, s.Init(rec, req, 0))
			require.NoError(t, s.JSON(http.StatusOK, json.RawMessage(body)))
			require.Empty(t, rec.Header().Get("Content-Encoding"))
			require.Empty(t, rec.Header().Get("Vary"))
			require.Equal(t, body, rec.Body.Bytes())
		})
	}
}

func TestCompressionPreservesLoggedJSONAcrossRequests(t *testing.T) {
	for _, encoding := range []string{"gzip", "zstd"} {
		t.Run(encoding, func(t *testing.T) {
			recorder := &requestRecorder{}
			server, err := zen.New(zen.Config{})
			require.NoError(t, err)
			body := jsonBody(t, 4096)
			server.RegisterRoute([]zen.Middleware{zen.WithMetrics(recorder, zen.InstanceInfo{}, nil), zen.WithCompression()},
				zen.NewRoute(http.MethodPost, "/analytics", func(_ context.Context, s *zen.Session) error {
					s.AddHeader("Content-Type", "application/json")
					return s.Send(http.StatusOK, body)
				}),
			)
			originals := make([]string, 0, 3)
			for i := range 3 {
				body, err = json.Marshal(struct{ Message string }{Message: strings.Repeat(fmt.Sprintf("row-%d", i), 1024)})
				require.NoError(t, err)
				originals = append(originals, string(body))
				req := httptest.NewRequest(http.MethodPost, "/analytics", nil)
				req.Header.Set("Accept-Encoding", encoding)
				rec := httptest.NewRecorder()
				server.Mux().ServeHTTP(rec, req)
				require.Equal(t, encoding, rec.Header().Get("Content-Encoding"))
				require.Equal(t, body, decode(t, rec))
			}
			require.Len(t, recorder.requests, len(originals))
			for i, original := range originals {
				require.Equal(t, original, recorder.requests[i].ResponseBody)
				require.Equal(t, int32(http.StatusOK), recorder.requests[i].ResponseStatus)
			}
		})
	}
}

type requestRecorder struct {
	requests []schema.ApiRequest
}

func (r *requestRecorder) Buffer(request schema.ApiRequest) {
	r.requests = append(r.requests, request)
}

func sendCompressedJSON(t *testing.T, s *zen.Session, body []byte) error {
	t.Helper()
	return zen.WithCompression()(func(_ context.Context, s *zen.Session) error {
		s.AddHeader("Content-Type", "application/json")
		return s.Send(http.StatusOK, body)
	})(t.Context(), s)
}

func newSession(t *testing.T, headers http.Header) (*zen.Session, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/analytics", nil)
	req.Header = headers
	rec := httptest.NewRecorder()
	s := new(zen.Session)
	require.NoError(t, s.Init(rec, req, 0))
	return s, rec
}

func jsonBody(t *testing.T, size int) []byte {
	t.Helper()
	var body struct {
		Data string `json:"data"`
	}
	empty, err := json.Marshal(body)
	require.NoError(t, err)
	body.Data = strings.Repeat("a", size-len(empty))
	encoded, err := json.Marshal(body)
	require.NoError(t, err)
	require.Len(t, encoded, size)
	return encoded
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) []byte {
	t.Helper()
	switch rec.Header().Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reader.Close()) })
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		return body
	case "zstd":
		reader, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
		require.NoError(t, err)
		t.Cleanup(reader.Close)
		body, err := reader.DecodeAll(rec.Body.Bytes(), nil)
		require.NoError(t, err)
		return body
	default:
		return rec.Body.Bytes()
	}
}
