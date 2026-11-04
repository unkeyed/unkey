package workos

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// UserEmail returns the user's email address. A user that does not exist has
// none, so it returns an empty string and no error.
func (c *client) UserEmail(ctx context.Context, userID string) (string, error) {
	u, err := c.getUser(ctx, userID)
	if sErr, ok := errors.AsType[*statusError](err); ok && sErr.code == http.StatusNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return u.Email, nil
}

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
