package workos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultAPIBaseURL = "https://api.workos.com"

// MembershipClient reads organization memberships from the WorkOS User
// Management API. CLI refresh uses it to learn the member's current roles.
type MembershipClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewMembershipClient returns a client for https://api.workos.com when baseURL is empty.
func NewMembershipClient(apiKey string, baseURL string) *MembershipClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultAPIBaseURL
	}
	return &MembershipClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

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
func (c *MembershipClient) ActiveRoles(ctx context.Context, orgID string, userID string) ([]string, bool, error) {
	query := url.Values{}
	query.Set("organization_id", orgID)
	query.Set("user_id", userID)
	query.Set("statuses", "active")
	query.Set("limit", "10")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/user_management/organization_memberships?"+query.Encode(), nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = res.Body.Close() }()

	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, false, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, false, fmt.Errorf("workos memberships: status %d", res.StatusCode)
	}

	var page membershipPage
	if err := json.Unmarshal(payload, &page); err != nil {
		return nil, false, fmt.Errorf("workos memberships: decode response: %w", err)
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
