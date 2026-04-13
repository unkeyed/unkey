package zen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

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
