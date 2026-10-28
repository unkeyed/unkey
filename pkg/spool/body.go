package spool

import (
	"errors"
	"io"
	"os"
	"sync"
)

// Body is a validated body stored in an unnamed temporary file. It holds one
// slot and Size bytes of its [Pool] until Close.
type Body struct {
	file  *os.File
	size  int64
	close func() error
}

func newBody(p *Pool, file *os.File, size int64) *Body {
	return &Body{
		file: file,
		size: size,
		close: sync.OnceValue(func() error {
			return errors.Join(closeFile(file), p.release(size))
		}),
	}
}

// Size returns the number of bytes stored, which equals the declared content
// length when one was given.
func (b *Body) Size() int64 {
	return b.size
}

// Reader returns a new, independent reader over the whole body, starting at
// offset 0. Readers never close the file, so a reader can be given to code
// that closes its request body. A reader is valid only until Close: after
// Close, a read fails or, for a reader at its end, returns io.EOF.
func (b *Body) Reader() io.Reader {
	return io.NewSectionReader(b.file, 0, b.size)
}

// Close closes the file and returns the slot and bytes to the pool. It is safe
// to call more than once and from many goroutines; every call waits for the
// first and returns its result.
func (b *Body) Close() error {
	return b.close()
}
