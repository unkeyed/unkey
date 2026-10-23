package zen

import (
	"bytes"
	"sync"

	"github.com/klauspost/compress/gzip"
	"github.com/unkeyed/unkey/pkg/fault"
)

var gzipEncoders sync.Pool

type gzipEncoder struct {
	writer *gzip.Writer
	buffer bytes.Buffer
}

func writeGzip(s *Session, body []byte) error {
	var encoder *gzipEncoder
	if cached := gzipEncoders.Get(); cached != nil {
		encoder = cached.(*gzipEncoder)
	} else {
		writer, err := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		if err != nil {
			return fault.Wrap(err, fault.Internal("create gzip encoder"))
		}
		encoder = &gzipEncoder{writer: writer, buffer: bytes.Buffer{}}
	}
	defer gzipEncoders.Put(encoder)

	encoder.buffer.Reset()
	encoder.writer.Reset(&encoder.buffer)
	if _, err := encoder.writer.Write(body); err != nil {
		return fault.Wrap(err, fault.Internal("compress response"))
	}
	if err := encoder.writer.Close(); err != nil {
		return fault.Wrap(err, fault.Internal("finish gzip response"))
	}
	return writeEncoded(s, body, encoder.buffer.Bytes(), "gzip")
}
