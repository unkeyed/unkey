package workos

import (
	"context"
	"net/url"
)

// getUser fetches a single user by id.
func (c *client) getUser(ctx context.Context, userID string) (user, error) {
	var u user
	err := c.get(ctx, "/user_management/users/"+url.PathEscape(userID), nil, &u)
	return u, err
}

// user is the slice of GET /user_management/users/{id} the resolver reads.
type user struct {
	Email string `json:"email"`
}
