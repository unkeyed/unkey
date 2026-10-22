package slack

import "context"

type noop struct{}

// NewNoop returns a client that discards Slack messages.
func NewNoop() noop { return noop{} }

func (noop) Send(context.Context, string, Payload) error { return nil }
