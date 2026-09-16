package cdc

import (
	"context"
	"errors"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
)

// Client watches one set of rules and remembers its last safe resume token.
// Use separate clients for separate consumers. Methods must not run concurrently.
type Client struct {
	connection *Connection
	rules      []Rule
	token      []byte
}

// New checks and copies the rules without opening a stream.
// The first Watch starts a snapshot. Later calls reuse the client's progress.
// It returns nil on error. The caller owns Connection, which must not be nil.
func New(cfg Config) (*Client, error) {
	if err := assert.NotNil(cfg.Connection, "CDC connection is required"); err != nil {
		return nil, err
	}
	if _, err := vstreamFilter(cfg.Rules); err != nil {
		return nil, err
	}
	return &Client{connection: cfg.Connection, rules: slices.Clone(cfg.Rules), token: nil}, nil
}

// Watch calls apply for FIELD and ROW changes, one at a time, and saves checkpoints
// internally. The callback must be non-nil, must not edit events, and must finish
// applying each change before returning. Repeated changes must be safe to apply.
// Errors stop the watch. Calling Watch again resumes from the last safe token.
// It does not retry automatically. After [ErrExpired], the next call starts a
// snapshot. The caller must remove destination records that no longer exist.
func (c *Client) Watch(ctx context.Context, apply func(*binlog.VEvent) error) error {
	err := c.connection.Forward(ctx, c.rules, c.token, func(event Event) error {
		if event.Change != nil {
			return apply(event.Change)
		}
		c.token = event.ResumeToken
		return nil
	})
	if errors.Is(err, ErrExpired) {
		c.token = nil
	}
	return err
}
