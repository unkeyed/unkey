package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
	"github.com/unkeyed/unkey/pkg/spool"
)

func TestHTTPValidationBeforeForward(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the spool prototype requires Linux")
	}

	const bodyBytesMax = 32 << 20
	pool, err := spool.NewPool(spool.Config{
		Dir:           t.TempDir(),
		BodyBytesMax:  bodyBytesMax,
		TotalBytesMax: bodyBytesMax,
		ConcurrentMax: 1,
	})
	require.NoError(t, err)
	schema := uploadSchema(t, bodyBytesMax*2)

	var upstreamRequests atomic.Int64
	var upstreamBytes atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)
		digest := sha256.New()
		n, readErr := io.Copy(digest, r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusBadRequest)
			return
		}
		upstreamBytes.Add(n)
		if _, writeErr := io.WriteString(w, hex.EncodeToString(digest.Sum(nil))); writeErr != nil {
			t.Errorf("write upstream digest: %v", writeErr)
		}
	}))
	t.Cleanup(upstream.Close)
	upstreamURL, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)

	completed := make(chan error, 1)
	frontend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var closeErr error
		defer func() { completed <- closeErr }()
		body, admitErr := pool.Admit(r.Context(), r.Body, r.ContentLength, schema.Validate)
		if admitErr != nil {
			status := http.StatusServiceUnavailable
			var failure *spool.Error
			if errors.As(admitErr, &failure) {
				if failure.Kind == spool.FailureInvalid {
					status = http.StatusBadRequest
				} else if failure.Kind == spool.FailureTooLarge {
					status = http.StatusRequestEntityTooLarge
				}
			}
			http.Error(w, admitErr.Error(), status)
			return
		}
		defer func() { closeErr = body.Close() }()
		r.Body = io.NopCloser(body.Reader())
		r.ContentLength = body.Size()
		r.TransferEncoding = nil
		proxy.ServeHTTP(w, r)
	}))
	frontend.Config.ReadTimeout = 30 * time.Second
	frontend.Start()
	t.Cleanup(frontend.Close)

	for _, test := range []struct {
		name       string
		imageBytes int64
		count      int
		chunked    bool
		status     int
	}{
		{name: "large known length", imageBytes: 16 << 20, count: 1, chunked: false, status: http.StatusOK},
		{name: "large chunked", imageBytes: 16 << 20, count: 1, chunked: true, status: http.StatusOK},
		{name: "late schema failure", imageBytes: 16 << 20, count: 0, chunked: true, status: http.StatusBadRequest},
		{name: "oversized chunked", imageBytes: bodyBytesMax + 1, count: 1, chunked: true, status: http.StatusRequestEntityTooLarge},
		{name: "capacity recovered after rejection", imageBytes: 11 << 20, count: 1, chunked: false, status: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			requestsBefore := upstreamRequests.Load()
			bytesBefore := upstreamBytes.Load()
			reader, size := uploadBody(t, test.imageBytes, test.count)
			req, requestErr := http.NewRequestWithContext(t.Context(), http.MethodPost, frontend.URL, reader)
			require.NoError(t, requestErr)
			req.Header.Set("Content-Type", "application/json")
			if !test.chunked {
				req.ContentLength = size
			}
			response, responseErr := frontend.Client().Do(req)
			require.NoError(t, responseErr)
			responseBody, readErr := io.ReadAll(response.Body)
			require.NoError(t, response.Body.Close())
			require.NoError(t, readErr)
			require.NoError(t, <-completed)
			require.Equal(t, test.status, response.StatusCode, string(responseBody))
			if test.status != http.StatusOK {
				require.Equal(t, requestsBefore, upstreamRequests.Load(), "rejected body reached upstream")
				return
			}
			require.Equal(t, requestsBefore+1, upstreamRequests.Load())
			require.Equal(t, size, upstreamBytes.Load()-bytesBefore)
			expected, _ := uploadBody(t, test.imageBytes, test.count)
			digest := sha256.New()
			_, hashErr := io.Copy(digest, expected)
			require.NoError(t, hashErr)
			require.Equal(t, hex.EncodeToString(digest.Sum(nil)), string(responseBody))
		})
	}
}

func TestValidationCancellationIsNotInvalidInput(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the spool prototype requires Linux")
	}
	pool, err := spool.NewPool(spool.Config{
		Dir:           t.TempDir(),
		BodyBytesMax:  1024,
		TotalBytesMax: 1024,
		ConcurrentMax: 1,
	})
	require.NoError(t, err)
	schema := uploadSchema(t, 1024)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	reader, size := uploadBody(t, 8, 1)
	body, err := pool.Admit(ctx, io.NopCloser(reader), size, func(ctx context.Context, r io.Reader) error {
		cancel()
		return schema.Validate(ctx, r)
	})
	require.Nil(t, body)
	require.ErrorIs(t, err, context.Canceled)
	var failure *spool.Error
	require.ErrorAs(t, err, &failure)
	require.Equal(t, spool.FailureCanceled, failure.Kind)

	reader, size = uploadBody(t, 8, 1)
	body, err = pool.Admit(t.Context(), io.NopCloser(reader), size, schema.Validate)
	require.NoError(t, err)
	require.NoError(t, body.Close())
}

func uploadSchema(t testing.TB, bytesMax int64) *streamschema.Schema {
	t.Helper()
	document, err := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"image": map[string]any{"type": "string", "minLength": 1},
			"count": map[string]any{"type": "integer", "minimum": 1},
		},
		"required":             []string{"image", "count"},
		"additionalProperties": false,
	})
	require.NoError(t, err)
	limits := streamschema.DefaultLimits()
	limits.InputBytesMax = bytesMax
	schema, err := streamschema.Compile(document, limits)
	require.NoError(t, err)
	return schema
}

// Fixed framing lets the test generate a large value without allocating that value.
func uploadBody(t testing.TB, imageBytes int64, count int) (io.Reader, int64) {
	t.Helper()
	tail, err := json.Marshal(struct {
		Count int `json:"count"`
	}{Count: count})
	require.NoError(t, err)
	prefix := `{"image":"`
	suffix := `",` + string(tail[1:])
	return io.MultiReader(
		strings.NewReader(prefix),
		io.LimitReader(repeatedASCII{}, imageBytes),
		strings.NewReader(suffix),
	), int64(len(prefix)+len(suffix)) + imageBytes
}

type repeatedASCII struct{}

func (repeatedASCII) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	return len(p), nil
}
