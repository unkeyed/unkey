package workos

import "context"

type noop struct{}

var _ Resolver = noop{}

// NewNoop returns a client that resolves no admin or member recipients.
func NewNoop() noop { return noop{} }

func (noop) AdminEmails(_ context.Context, _ string) ([]string, error) { return nil, nil }

func (noop) MemberEmails(_ context.Context, _ string) ([]string, error) { return nil, nil }

func (noop) ActiveRoles(context.Context, string, string) ([]string, bool, error) {
	return nil, false, nil
}

func (noop) DeleteUser(context.Context, string) (bool, error) { return false, nil }

func (noop) DeleteOrganization(context.Context, string) (bool, error) { return false, nil }
