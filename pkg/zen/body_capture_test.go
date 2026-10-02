package zen

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBodyCaptureBoundsStorageWithoutShortWrites(t *testing.T) {
	for _, size := range []int{0, 5, 257 << 10, MaxBodyCapture, MaxBodyCapture + 777} {
		for _, chunkSize := range []int{17 << 10, 32 << 10, MaxBodyCapture + 1} {
			for _, hint := range []int64{-1, 1, int64(size), 1 << 40} {
				t.Run(fmt.Sprintf("size=%d/chunk=%d/hint=%d", size, chunkSize, hint), func(t *testing.T) {
					capture := NewBodyCapture(hint)
					payload := bytes.Repeat([]byte("x"), size)
					for start := 0; start < len(payload); start += chunkSize {
						chunk := payload[start:min(start+chunkSize, len(payload))]
						n, err := capture.Write(chunk)
						require.NoError(t, err)
						require.Equal(t, len(chunk), n)
						require.LessOrEqual(t, cap(capture.Bytes()), MaxBodyCapture)
					}
					require.Equal(t, string(payload[:min(size, MaxBodyCapture)]), string(capture.Bytes()))
				})
			}
		}
	}
}

func TestBodyCaptureDoesNotReserveDeclaredSizeBeforeReceivingBytes(t *testing.T) {
	capture := NewBodyCapture(1 << 40)
	require.Nil(t, capture.Bytes())
	n, err := capture.Write([]byte("x"))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, 64, cap(capture.Bytes()))
}

func TestBodyCaptureOwnsBytesAcrossRequests(t *testing.T) {
	var first BodyCapture
	source := []byte("first request")
	_, err := first.Write(source)
	require.NoError(t, err)
	saved := first.Bytes()
	clear(source)
	for range 100 {
		capture := NewBodyCapture(13)
		_, err = capture.Write([]byte("other request"))
		require.NoError(t, err)
	}
	require.Equal(t, "first request", string(saved))
}

func BenchmarkBodyCapture(b *testing.B) {
	for _, size := range []int{257 << 10, MaxBodyCapture} {
		for _, mode := range []string{"legacy", "known-length", "unknown-length"} {
			b.Run(fmt.Sprintf("%d/%s", size, mode), func(b *testing.B) {
				payload := bytes.Repeat([]byte("x"), size)
				capacity := 0
				b.ReportAllocs()
				b.SetBytes(int64(size))
				for b.Loop() {
					var capture interface {
						io.Writer
						Bytes() []byte
					}
					switch mode {
					case "legacy":
						capture = &legacyBodyCapture{}
					case "known-length":
						capture = NewBodyCapture(int64(size))
					case "unknown-length":
						capture = NewBodyCapture(-1)
					}
					for start := 0; start < size; start += 17 << 10 {
						_, err := capture.Write(payload[start:min(start+17<<10, size)])
						if err != nil {
							b.Fatal(err)
						}
					}
					capacity = cap(capture.Bytes())
				}
				b.ReportMetric(float64(capacity), "capacity-B")
			})
		}
	}
}

type legacyBodyCapture struct{ bytes.Buffer }

func (c *legacyBodyCapture) Write(p []byte) (int, error) {
	_, err := c.Buffer.Write(p[:min(len(p), MaxBodyCapture-c.Len())])
	return len(p), err
}
