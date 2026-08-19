package workos

import "context"

// noop resolves no recipients. Wired by callers that have no WorkOS key
// configured, so the spend check still runs and logs but sends no email.
type noop struct{}

var _ Resolver = noop{}

// NewNoop returns a Resolver that resolves no recipients.
func NewNoop() Resolver { return noop{} }

func (noop) AdminEmails(_ context.Context, _ string) ([]string, error) { return nil, nil }
