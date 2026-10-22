package workos

import "context"

type noop struct{}

var _ Resolver = noop{}

// NewNoop returns a client that resolves no admin or member recipients.
func NewNoop() noop { return noop{} }

func (noop) AdminEmails(_ context.Context, _ string) ([]string, error) { return nil, nil }

func (noop) MemberEmails(_ context.Context, _ string) ([]string, error) { return nil, nil }
