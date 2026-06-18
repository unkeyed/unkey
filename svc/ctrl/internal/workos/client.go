package workos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
)

// baseURL is the WorkOS REST API origin.
const baseURL = "https://api.workos.com"

// adminRoleSlug is the membership role that receives billing alerts; matches the
// dashboard's own admin check.
const adminRoleSlug = "admin"

// membershipPageSize bounds each org-memberships page.
const membershipPageSize = 100

// requestTimeout bounds each WorkOS call. The lookup runs inside a journaled
// Restate step on a budget-threshold crossing; a hung request should fail the
// step and retry rather than stall the invocation.
const requestTimeout = 30 * time.Second

// client is the WorkOS-backed Resolver.
type client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

var _ Resolver = (*client)(nil)

// New returns a WorkOS-backed Resolver. apiKey must be non-empty; a caller
// with no key configured wires NewNoop instead and owns that decision.
func New(apiKey string) Resolver {
	return &client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: requestTimeout}, //nolint:exhaustruct // default transport, only the timeout matters
	}
}

// AdminEmails pages the org's active memberships and resolves the email of each
// admin. Admin counts are tiny and this runs only on a budget-threshold
// crossing, so the per-admin user lookup is cheap.
func (c *client) AdminEmails(ctx context.Context, orgID string) ([]string, error) {
	var emails []string
	after := ""
	for {
		page, err := c.listOrganizationMemberships(ctx, orgID, after)
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("list workos org memberships"))
		}

		for _, m := range page.Data {
			if m.Role.Slug != adminRoleSlug {
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

// getUser fetches a single user by id.
func (c *client) getUser(ctx context.Context, userID string) (user, error) {
	var u user
	err := c.get(ctx, "/user_management/users/"+url.PathEscape(userID), nil, &u)
	return u, err
}

// get performs an authenticated GET and decodes the JSON response into out. A
// non-2xx status is an error carrying a bounded slice of the body, which is
// where WorkOS puts its error code and message.
func (c *client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("workos GET %s: status %d: %s", path, res.StatusCode, body)
	}
	return json.NewDecoder(res.Body).Decode(out)
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

// user is the slice of GET /user_management/users/{id} the resolver reads.
type user struct {
	Email string `json:"email"`
}
