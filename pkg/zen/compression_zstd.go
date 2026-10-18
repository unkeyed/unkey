package zen

import (
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/unkeyed/unkey/pkg/fault"
)

var zstdEncoders sync.Pool

type zstdEncoder struct {
	writer *zstd.Encoder
	buffer []byte
}

func writeZstd(s *Session, body []byte) error {
	var encoder *zstdEncoder
	if cached := zstdEncoders.Get(); cached != nil {
		encoder = cached.(*zstdEncoder)
	} else {
		writer, err := zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.SpeedFastest),
			zstd.WithEncoderConcurrency(1),
		)
		if err != nil {
			return fault.Wrap(err, fault.Internal("create zstd encoder"))
		}
		encoder = &zstdEncoder{writer: writer, buffer: nil}
	}
	defer zstdEncoders.Put(encoder)

	encoder.buffer = encoder.writer.EncodeAll(body, encoder.buffer[:0])
	return writeEncoded(s, body, encoder.buffer, "zstd")
}
