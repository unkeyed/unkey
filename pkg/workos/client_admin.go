package workos

import (
	"context"

	"github.com/unkeyed/unkey/pkg/fault"
)

// AdminEmails pages the org's active memberships and resolves the email of each
// admin. Admin counts are tiny and this runs only on a budget-threshold
// crossing, so the per-admin user lookup is cheap.
func (c *client) AdminEmails(ctx context.Context, orgID string) ([]string, error) {
	return c.membershipEmails(ctx, orgID, adminRoleSlug)
}

// MemberEmails returns unique, nonempty active member emails in encounter order.
func (c *client) MemberEmails(ctx context.Context, orgID string) ([]string, error) {
	emails, err := c.membershipEmails(ctx, orgID, "")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(emails))
	var unique []string
	for _, email := range emails {
		if !seen[email] {
			seen[email] = true
			unique = append(unique, email)
		}
	}
	return unique, nil
}

func (c *client) membershipEmails(ctx context.Context, orgID, roleSlug string) ([]string, error) {
	var emails []string
	after := ""
	for {
		page, err := c.listOrganizationMemberships(ctx, orgID, after)
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("list workos org memberships"))
		}

		for _, m := range page.Data {
			if roleSlug != "" && m.Role.Slug != roleSlug {
				continue
			}
			u, err := c.getUser(ctx, m.UserID)
			if err != nil {
				return nil, fault.Wrap(err, fault.Internal("get workos user"))
			}
			if u.Email != "" {
				emails = append(emails, u.Email)
			}
		}

		if page.ListMetadata.After == "" {
			return emails, nil
		}
		after = page.ListMetadata.After
	}
}
