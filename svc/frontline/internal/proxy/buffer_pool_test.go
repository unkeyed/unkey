package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyBufferPoolClearsAndBoundsIdleBuffers(t *testing.T) {
	pool := make(copyBufferPool, 1)
	first, second := pool.Get(), pool.Get()
	require.Len(t, first, copyBufferSize)
	require.Len(t, second, copyBufferSize)
	copy(first, "first tenant")
	copy(second, "second tenant")
	pool.Put(first)
	pool.Put(second)
	require.Len(t, pool, 1)
	require.Equal(t, make([]byte, copyBufferSize), pool.Get())
	require.Empty(t, pool)
	require.Equal(t, make([]byte, copyBufferSize), second)
}

func BenchmarkReverseProxyBufferPool(b *testing.B) {
	for _, pooled := range []bool{false, true} {
		name := "unpooled"
		if pooled {
			name = "pooled"
		}
		b.Run(name, func(b *testing.B) {
			payload := strings.Repeat("x", 64<<10)
			proxy := httputil.ReverseProxy{
				Director:  func(_ *http.Request) {},
				Transport: bufferBenchmarkTransport{payload: payload},
			}
			if pooled {
				proxy.BufferPool = make(copyBufferPool, 64)
			}
			req := httptest.NewRequest(http.MethodGet, "http://upstream.test/", nil)
			b.ReportAllocs()
			b.SetBytes(int64(len(payload)))
			b.ResetTimer()
			for b.Loop() {
				proxy.ServeHTTP(&discardResponseWriter{header: make(http.Header)}, req)
			}
		})
	}
}

type bufferBenchmarkTransport struct{ payload string }

func (t bufferBenchmarkTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(t.payload)),
		ContentLength: int64(len(t.payload)),
	}, nil
}

type discardResponseWriter struct{ header http.Header }

func (w *discardResponseWriter) Header() http.Header       { return w.header }
func (*discardResponseWriter) Write(p []byte) (int, error) { return len(p), nil }
func (*discardResponseWriter) WriteHeader(_ int)           {}
