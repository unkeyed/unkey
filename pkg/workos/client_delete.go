package workos

import (
	"context"
	"net/url"
)

// DeleteUser deletes a WorkOS user and reports whether it existed. A user that
// does not exist counts as deleted.
func (c *client) DeleteUser(ctx context.Context, userID string) (bool, error) {
	return c.delete(ctx, "/user_management/users/"+url.PathEscape(userID))
}

// DeleteOrganization deletes a WorkOS organization and reports whether it
// existed. An organization that does not exist counts as deleted.
func (c *client) DeleteOrganization(ctx context.Context, orgID string) (bool, error) {
	return c.delete(ctx, "/organizations/"+url.PathEscape(orgID))
}
