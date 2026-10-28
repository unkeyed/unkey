// Package workos reads organization roles and member email addresses, and
// deletes users and organizations for GDPR erasure.
// New selects a noop when no API key is configured.
package workos

import "context"

// Client resolves both admin and member recipients for an organization.
type Client interface {
	Resolver
	MemberEmails(ctx context.Context, orgID string) ([]string, error)
	ActiveRoles(ctx context.Context, orgID string, userID string) ([]string, bool, error)
	Eraser
}

// Eraser deletes WorkOS records for GDPR erasure and reports whether each
// record existed. Deleting a record that does not exist succeeds, so a repeated
// erasure converges.
type Eraser interface {
	DeleteUser(ctx context.Context, userID string) (bool, error)
	DeleteOrganization(ctx context.Context, orgID string) (bool, error)
}

// Resolver returns the email addresses to notify for an organization.
type Resolver interface {
	// AdminEmails returns the active admins' emails for an org. An empty slice
	// (no admins, or a noop resolver) means "send to no one": a skip, not an
	// error.
	AdminEmails(ctx context.Context, orgID string) ([]string, error)
}
