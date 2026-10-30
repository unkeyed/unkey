package workos

import (
	"context"
	"net/url"
	"strings"
)

type membershipPage struct {
	Data []struct {
		Role struct {
			Slug string `json:"slug"`
		} `json:"role"`
		Roles []struct {
			Slug string `json:"slug"`
		} `json:"roles"`
	} `json:"data"`
}

// ActiveRoles returns the role slugs on the user's active membership in the
// organization. found is false when WorkOS has no active membership. A non-nil
// error means the lookup itself failed.
func (c *client) ActiveRoles(ctx context.Context, orgID string, userID string) ([]string, bool, error) {
	query := url.Values{}
	query.Set("organization_id", orgID)
	query.Set("user_id", userID)
	query.Set("statuses", "active")
	query.Set("limit", "10")

	var page membershipPage
	if err := c.get(ctx, "/user_management/organization_memberships", query, &page); err != nil {
		return nil, false, err
	}
	if len(page.Data) == 0 {
		return nil, false, nil
	}

	seen := map[string]struct{}{}
	var roles []string
	for _, membership := range page.Data {
		if slug := strings.TrimSpace(membership.Role.Slug); slug != "" {
			if _, ok := seen[slug]; !ok {
				seen[slug] = struct{}{}
				roles = append(roles, slug)
			}
		}
		for _, role := range membership.Roles {
			slug := strings.TrimSpace(role.Slug)
			if slug == "" {
				continue
			}
			if _, ok := seen[slug]; ok {
				continue
			}
			seen[slug] = struct{}{}
			roles = append(roles, slug)
		}
	}
	return roles, true, nil
}
