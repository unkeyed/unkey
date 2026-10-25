package workos

import (
	"context"
	"net/url"
	"strconv"
)

// getUser fetches a single user by id.
func (c *client) getUser(ctx context.Context, userID string) (user, error) {
	var u user
	err := c.get(ctx, "/user_management/users/"+url.PathEscape(userID), nil, &u)
	return u, err
}

// user is the slice of GET /user_management/users/{id} the resolver reads.
type user struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type usersPage struct {
	Data         []user       `json:"data"`
	ListMetadata listMetadata `json:"list_metadata"`
}

func (c *client) listOrganizationUsers(ctx context.Context, orgID, after string) (usersPage, error) {
	query := url.Values{"organization_id": {orgID}, "limit": {strconv.Itoa(membershipPageSize)}}
	if after != "" {
		query.Set("after", after)
	}
	var page usersPage
	err := c.get(ctx, "/user_management/users", query, &page)
	return page, err
}
