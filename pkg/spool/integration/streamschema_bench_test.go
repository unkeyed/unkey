package integration_test

import (
	"fmt"
	"io"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/spool"
)

func BenchmarkValidateSpoolReplay(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("the spool prototype requires Linux")
	}
	for _, imageBytes := range []int64{1 << 20, 16 << 20, 64 << 20} {
		b.Run(fmt.Sprintf("%dMiB", imageBytes>>20), func(b *testing.B) {
			const bodyBytesMax = 128 << 20
			pool, err := spool.NewPool(spool.Config{
				Dir:           b.TempDir(),
				BodyBytesMax:  bodyBytesMax,
				TotalBytesMax: bodyBytesMax,
				ConcurrentMax: 1,
			})
			require.NoError(b, err)
			schema := uploadSchema(b, bodyBytesMax)
			buffer := make([]byte, 32<<10)
			b.ReportAllocs()
			b.SetBytes(imageBytes)
			for b.Loop() {
				reader, size := uploadBody(b, imageBytes, 1)
				body, admitErr := pool.Admit(b.Context(), io.NopCloser(reader), size, schema.Validate)
				require.NoError(b, admitErr)
				copied, copyErr := io.CopyBuffer(io.Discard, body.Reader(), buffer)
				closeErr := body.Close()
				require.NoError(b, copyErr)
				require.NoError(b, closeErr)
				require.Equal(b, size, copied)
			}
		})
	}
}
