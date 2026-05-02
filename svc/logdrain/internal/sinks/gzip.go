package sinks

import (
	"bytes"
	"compress/gzip"
)

// GzipBytes compresses the input with the default gzip level. Axiom accepts
// gzip via `Content-Encoding: gzip`, and the bandwidth saved on a 1–5 MB JSON
// batch easily pays for the per-batch CPU. The compressor pool is
// intentionally not shared: gzip.Writer holds a ~64 KB internal buffer,
// and per-batch allocation is cheaper than the lock contention a shared
// pool would introduce on the per-tick fan-out goroutines.
func GzipBytes(in []byte) ([]byte, error) {
	var buf bytes.Buffer
	// Pre-size the buffer assuming ~30% compression on JSON; off by a few
	// percent in either direction is fine, this only avoids one or two
	// growth reallocations on the typical batch.
	buf.Grow(len(in) * 3 / 10)
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(in); err != nil {
		_ = w.Close()
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
