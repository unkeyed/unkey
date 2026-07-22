package email

import (
	"context"
	"sync"
)

// Capture is a Sender for tests: it records delivered emails and simulates the
// provider's idempotency-key dedup, so a test can distinguish an email never
// attempted from one attempted but deduped. Safe for concurrent use.
type Capture struct {
	mu   sync.Mutex
	sent []Email
	seen map[string]struct{}
}

// NewCapture returns an empty capturing sender.
func NewCapture() *Capture {
	return &Capture{
		mu:   sync.Mutex{},
		sent: nil,
		seen: make(map[string]struct{}),
	}
}

func (c *Capture) Send(_ context.Context, e Email) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e.IdempotencyKey != "" {
		if _, dup := c.seen[e.IdempotencyKey]; dup {
			return nil
		}
		c.seen[e.IdempotencyKey] = struct{}{}
	}
	c.sent = append(c.sent, e)
	return nil
}

// Sent returns a copy of the emails delivered so far in send order, excluding
// sends dropped by dedup.
func (c *Capture) Sent() []Email {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Email, len(c.sent))
	copy(out, c.sent)
	return out
}

// CountByTemplate returns how many emails of a template were delivered.
func (c *Capture) CountByTemplate(templateID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, e := range c.sent {
		if e.TemplateID == templateID {
			n++
		}
	}
	return n
}
