//go:build linux

package integration_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
)

func comparisonMemoryAdmission(schema *streamschema.Schema, bodyBytesMax, totalBytesMax, concurrentMax int64) func(*http.Request) (io.ReadCloser, error) {
	var mu sync.Mutex
	var reserved, active int64
	return func(req *http.Request) (result io.ReadCloser, err error) {
		var inputCloseErr error
		closeInput := sync.OnceFunc(func() { inputCloseErr = req.Body.Close() })
		stop := context.AfterFunc(req.Context(), closeInput)
		var body *comparisonMemoryBody
		defer func() {
			closeInput()
			stop()
			err = errors.Join(err, inputCloseErr, req.Context().Err())
			if err != nil && body != nil {
				err = errors.Join(err, body.Close())
				result = nil
			}
		}()
		if req.ContentLength < -1 || req.ContentLength > bodyBytesMax {
			return nil, errors.New("body length exceeds comparison limits")
		}
		limit := bodyBytesMax
		if req.ContentLength >= 0 {
			limit = req.ContentLength
		}
		mu.Lock()
		if active >= concurrentMax || limit > totalBytesMax-reserved {
			mu.Unlock()
			return nil, errors.New("memory comparison capacity exhausted")
		}
		active++
		reserved += limit
		mu.Unlock()
		body = &comparisonMemoryBody{limit: limit}
		body.close = sync.OnceFunc(func() {
			body.mu.Lock()
			body.chunks = nil
			body.mu.Unlock()
			mu.Lock()
			reserved -= limit
			active--
			mu.Unlock()
		})
		reader := &io.LimitedReader{R: req.Body, N: limit + 1}
		validateErr := schema.Validate(req.Context(), io.TeeReader(reader, body))
		if validateErr != nil {
			var invalid *streamschema.ValidationError
			if errors.As(validateErr, &invalid) {
				return nil, errors.Join(errRejected, validateErr)
			}
			return nil, validateErr
		}
		if req.ContentLength >= 0 && body.size != req.ContentLength {
			return nil, errors.New("body length differs from declared length")
		}
		return body, nil
	}
}

type comparisonMemoryBody struct {
	mu     sync.Mutex
	chunks [][]byte
	size   int64
	limit  int64
	index  int
	offset int
	close  func()
}

func (b *comparisonMemoryBody) Write(p []byte) (int, error) {
	if int64(len(p)) > b.limit-b.size {
		return 0, errors.New("body exceeds comparison byte limit")
	}
	n := len(p)
	for len(p) > 0 {
		if len(b.chunks) == 0 || len(b.chunks[len(b.chunks)-1]) == cap(b.chunks[len(b.chunks)-1]) {
			b.chunks = append(b.chunks, make([]byte, 0, min(32<<10, b.limit-b.size)))
		}
		i := len(b.chunks) - 1
		count := min(len(p), cap(b.chunks[i])-len(b.chunks[i]))
		b.chunks[i] = append(b.chunks[i], p[:count]...)
		b.size += int64(count)
		p = p[count:]
	}
	return n, nil
}

func (b *comparisonMemoryBody) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	if b.index == len(b.chunks) || len(b.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.chunks[b.index][b.offset:])
	b.offset += n
	if b.offset == len(b.chunks[b.index]) {
		b.index++
		b.offset = 0
	}
	return n, nil
}

func (b *comparisonMemoryBody) Close() error {
	b.close()
	return nil
}

func TestMemoryComparisonAdmission(t *testing.T) {
	schema := uploadSchema(t, 1024)
	request := func(imageBytes int64, count int) *http.Request {
		t.Helper()
		input, size := uploadBody(t, imageBytes, count)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://comparison/upload", input)
		require.NoError(t, err)
		req.ContentLength = size
		return req
	}
	for _, test := range []struct {
		name       string
		imageBytes int64
		count      int
		length     int64
		valid      bool
	}{
		{name: "known length", imageBytes: 8, count: 1, length: 30, valid: true},
		{name: "unknown length", imageBytes: 8, count: 1, length: -1, valid: true},
		{name: "exact limit", imageBytes: 42, count: 1, length: -1, valid: true},
		{name: "known oversized", imageBytes: 43, count: 1, length: 65, valid: false},
		{name: "unknown oversized", imageBytes: 43, count: 1, length: -1, valid: false},
		{name: "declared too short", imageBytes: 8, count: 1, length: 29, valid: false},
		{name: "declared too long", imageBytes: 8, count: 1, length: 31, valid: false},
		{name: "late invalid", imageBytes: 8, count: 0, length: 30, valid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			admit := comparisonMemoryAdmission(schema, 64, 64, 1)
			req := request(test.imageBytes, test.count)
			req.ContentLength = test.length
			body, err := admit(req)
			if test.valid {
				require.NoError(t, err)
				expected, _ := uploadBody(t, test.imageBytes, test.count)
				want, readErr := io.ReadAll(expected)
				require.NoError(t, readErr)
				got, readErr := io.ReadAll(body)
				require.NoError(t, readErr)
				require.Equal(t, want, got)
				require.NoError(t, body.Close())
			} else {
				require.Error(t, err)
				require.Nil(t, body)
			}
			recovered, err := admit(request(8, 1))
			require.NoError(t, err)
			require.NoError(t, recovered.Close())
		})
	}
	for _, test := range []struct {
		name          string
		totalBytesMax int64
		concurrentMax int64
	}{
		{name: "byte reservation", totalBytesMax: 64, concurrentMax: 8},
		{name: "concurrency reservation", totalBytesMax: 128, concurrentMax: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			admit := comparisonMemoryAdmission(schema, 64, test.totalBytesMax, test.concurrentMax)
			first, err := admit(request(24, 1))
			require.NoError(t, err)
			rejected, err := admit(request(24, 1))
			require.Error(t, err)
			require.Nil(t, rejected)
			require.NoError(t, first.Close())
			second, err := admit(request(24, 1))
			require.NoError(t, err)
			require.NoError(t, first.Close())
			rejected, err = admit(request(24, 1))
			require.Error(t, err)
			require.Nil(t, rejected)
			require.NoError(t, second.Close())
		})
	}
}
