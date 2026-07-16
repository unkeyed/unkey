package workos

import (
	"context"
	"net/url"
	"strconv"
)

// listOrganizationMemberships fetches one page of the org's active
// memberships, starting after the given cursor; "" fetches the first page.
func (c *client) listOrganizationMemberships(ctx context.Context, orgID, after string) (membershipsPage, error) {
	q := url.Values{}
	q.Set("organization_id", orgID)
	q.Set("statuses", "active")
	q.Set("limit", strconv.Itoa(membershipPageSize))
	if after != "" {
		q.Set("after", after)
	}

	var page membershipsPage
	err := c.get(ctx, "/user_management/organization_memberships", q, &page)
	return page, err
}

// membershipsPage is the slice of GET /user_management/organization_memberships
// the resolver reads: the members with their roles, plus the pagination cursor.
type membershipsPage struct {
	Data         []membership `json:"data"`
	ListMetadata listMetadata `json:"list_metadata"`
}

// membership is one organization member: the user and the role that decides
// whether they receive billing alerts.
type membership struct {
	UserID string         `json:"user_id"`
	Role   membershipRole `json:"role"`
}

// membershipRole carries the role slug compared against adminRoleSlug.
type membershipRole struct {
	Slug string `json:"slug"`
}

// listMetadata is WorkOS's cursor envelope; an empty After is the last page.
type listMetadata struct {
	After string `json:"after"`
}
