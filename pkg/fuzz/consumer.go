package fuzz

import "testing"

// Consumer consumes bytes from fuzzer input to generate typed values.
//
// Consumer tracks a position within the input byte slice and advances it
// as values are extracted. When insufficient bytes remain for a requested
// operation, Consumer automatically calls t.Skip() to abort the test iteration.
//
// Consumer is not safe for concurrent use.
type Consumer struct {
	t    *testing.T
	data []byte
	pos  int
}

// New creates a Consumer from fuzzer-provided bytes.
//
// The Consumer will use t.Skip() to abort test iterations when the input
// is exhausted, ensuring all generated values come from fuzzer-controlled bytes.
func New(t *testing.T, data []byte) *Consumer {
	return &Consumer{
		t:    t,
		data: data,
		pos:  0,
	}
}

// Remaining returns the number of unconsumed bytes.
func (c *Consumer) Remaining() int {
	return len(c.data) - c.pos
}

// Exhausted returns true if no bytes remain to be consumed.
func (c *Consumer) Exhausted() bool {
	return c.pos >= len(c.data)
}

// skip aborts the current test iteration due to insufficient input.
// This is called internally when an extraction would exceed available bytes.
func (c *Consumer) skip() {
	c.t.Skip("fuzz: insufficient input bytes")
}

// take consumes n bytes from the input, skipping if insufficient bytes remain.
func (c *Consumer) take(n int) []byte {
	if c.pos+n > len(c.data) {
		c.skip()
	}
	start := c.pos
	c.pos += n
	return c.data[start:c.pos]
}

// takeByte consumes a single byte, skipping if no bytes remain.
func (c *Consumer) takeByte() byte {
	if c.pos >= len(c.data) {
		c.skip()
	}
	b := c.data[c.pos]
	c.pos++
	return b
}
