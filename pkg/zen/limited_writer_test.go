package zen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCaptureBufferHint(t *testing.T) {
	require.Equal(t, 0, CaptureBufferHint(-1), "unknown length must not pre-allocate")
	require.Equal(t, 0, CaptureBufferHint(0), "zero length must not pre-allocate")
	require.Equal(t, 1024, CaptureBufferHint(1024), "known small length sizes exactly")
	require.Equal(t, MaxBodyCapture, CaptureBufferHint(MaxBodyCapture), "at-cap length sizes to cap")
	require.Equal(t, MaxBodyCapture, CaptureBufferHint(MaxBodyCapture+1), "over-cap length clamps to cap")
}

func TestLimitedWriter_CapsAtLimit(t *testing.T) {
	var buf bytes.Buffer
	lw := &LimitedWriter{W: &buf, N: 10}

	n, err := lw.Write([]byte("hello world!")) // 12 bytes
	require.NoError(t, err)
	require.Equal(t, 12, n, "should report all bytes as written to avoid short-write errors")
	require.Equal(t, "hello worl", buf.String(), "should only capture first 10 bytes")
}

func TestLimitedWriter_ExactLimit(t *testing.T) {
	var buf bytes.Buffer
	lw := &LimitedWriter{W: &buf, N: 5}

	n, err := lw.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, "hello", buf.String())
}

func TestLimitedWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	lw := &LimitedWriter{W: &buf, N: 8}

	_, _ = lw.Write([]byte("aaa"))  // 3 written, 5 remaining
	_, _ = lw.Write([]byte("bbb"))  // 3 written, 2 remaining
	_, _ = lw.Write([]byte("cccc")) // only 2 captured, rest discarded

	require.Equal(t, "aaabbbcc", buf.String())
}

func TestLimitedWriter_ZeroLimit(t *testing.T) {
	var buf bytes.Buffer
	lw := &LimitedWriter{W: &buf, N: 0}

	n, err := lw.Write([]byte("anything"))
	require.NoError(t, err)
	require.Equal(t, 8, n, "should report all bytes as written")
	require.Empty(t, buf.String(), "should capture nothing")
}
