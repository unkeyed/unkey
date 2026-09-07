package zen

// BodyCapture copies at most MaxBodyCapture bytes without interrupting its source.
// Its zero value is ready to use. It must not be reused between requests.
type BodyCapture struct {
	body     []byte
	sizeHint int
}

// NewBodyCapture uses contentLength only as a capacity hint, not a read limit.
// Memory is allocated as bytes arrive, even when the declared length is large.
func NewBodyCapture(contentLength int64) *BodyCapture {
	return &BodyCapture{body: nil, sizeHint: int(min(max(contentLength, 0), MaxBodyCapture))}
}

// Write reports all bytes consumed, including bytes beyond the capture limit.
func (c *BodyCapture) Write(p []byte) (int, error) {
	n := min(len(p), MaxBodyCapture-len(c.body))
	needed := len(c.body) + n
	if needed > cap(c.body) {
		capacity := min(MaxBodyCapture, max(needed, 64, 2*cap(c.body)))
		if c.sizeHint >= needed {
			capacity = min(capacity, c.sizeHint)
		}
		body := make([]byte, len(c.body), capacity)
		copy(body, c.body)
		c.body = body
	}
	c.body = append(c.body, p[:n]...)
	return len(p), nil
}

// Bytes returns the captured bytes without copying. Stop writing before handing
// them to another owner; the capture never pools or reuses their backing array.
func (c *BodyCapture) Bytes() []byte {
	return c.body
}
