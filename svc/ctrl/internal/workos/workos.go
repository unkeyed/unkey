// Package workos resolves an organization's admin emails via the WorkOS API.
// Budget alerts go to workspace admins, whose emails live in WorkOS, not our
// database, so the spend-cap check has to look them up here.
package workos

import (
	"context"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/workos/workos-go/v4/pkg/usermanagement"
)

// adminRoleSlug is the membership role that receives billing alerts; matches the
// dashboard's own admin check.
const adminRoleSlug = "admin"

// membershipPageSize bounds each org-memberships page.
const membershipPageSize = 100

// Resolver returns the email addresses to notify for an organization.
type Resolver interface {
	// AdminEmails returns the active admins' emails for an org. An empty slice
	// (no admins, or a noop resolver) means "send to no one": a skip, not an
	// error.
	AdminEmails(ctx context.Context, orgID string) ([]string, error)
}

// New returns a WorkOS-backed Resolver, or a noop when apiKey is empty (the
// caller decides real-vs-noop by whether a key is configured).
func New(apiKey string) Resolver {
	if apiKey == "" {
		return noop{}
	}
	return &client{wm: usermanagement.NewClient(apiKey)}
}

type client struct {
	wm *usermanagement.Client
}

// AdminEmails pages the org's active memberships and resolves the email of each
// admin. Admin counts are tiny and this runs only on a budget-threshold
// crossing, so the per-admin user lookup is cheap.
func (c *client) AdminEmails(ctx context.Context, orgID string) ([]string, error) {
	var emails []string
	after := ""
	for {
		page, err := c.wm.ListOrganizationMemberships(ctx, usermanagement.ListOrganizationMembershipsOpts{ //nolint:exhaustruct // only filtering by org + active status; other filters left at defaults
			OrganizationID: orgID,
			Statuses:       []usermanagement.OrganizationMembershipStatus{usermanagement.Active},
			Limit:          membershipPageSize,
			After:          after,
		})
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("list workos org memberships"))
		}

		for _, m := range page.Data {
			if m.Role.Slug != adminRoleSlug {
				continue
			}
			user, err := c.wm.GetUser(ctx, usermanagement.GetUserOpts{User: m.UserID})
			if err != nil {
				return nil, fault.Wrap(err, fault.Internal("get workos user"))
			}
			if user.Email != "" {
				emails = append(emails, user.Email)
			}
		}

		if page.ListMetadata.After == "" {
			return emails, nil
		}
		after = page.ListMetadata.After
	}
}

// noop resolves no recipients. Used when no WorkOS key is configured, so the
// spend check still runs and logs but sends no email.
type noop struct{}

func (noop) AdminEmails(_ context.Context, _ string) ([]string, error) { return nil, nil }
