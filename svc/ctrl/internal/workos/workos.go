// Package workos resolves an organization's admin emails via the WorkOS API.
// Budget alerts go to workspace admins, whose emails live in WorkOS, not our
// database, so the spend-cap check has to look them up here.
//
// The two GET endpoints this needs do not justify the WorkOS SDK dependency,
// so the client is plain net/http against the documented REST API. Callers
// choose the implementation: New for the real client, NewNoop when no API key
// is configured (see noop.go).
package workos

import "context"

// Resolver returns the email addresses to notify for an organization.
type Resolver interface {
	// AdminEmails returns the active admins' emails for an org. An empty slice
	// (no admins, or a noop resolver) means "send to no one": a skip, not an
	// error.
	AdminEmails(ctx context.Context, orgID string) ([]string, error)
}
