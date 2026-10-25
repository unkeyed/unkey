package workos

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"sort"
	"strings"
)

const (
	maxDirectoryMembers = 1000
	maxDirectoryPages   = 100
)

// ErrDirectoryLimit stops exhausted or non-progressing directory traversal; don't retry unchanged.
var ErrDirectoryLimit = errors.New("workos directory limit exceeded")

// Member is an active organization member with a verified account email.
type Member struct {
	UserID string
	Email  string
	Admin  bool
}

type Directory interface {
	Members(ctx context.Context, orgID string) ([]Member, error)
	LookupMembers(ctx context.Context, orgID string, userIDs []string) ([]Member, error)
	CanEmail(ctx context.Context, orgID, userID, email string) (bool, error)
}

var _ Directory = (*client)(nil)

func NewDirectory(apiKey string) (Directory, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("workos API key is not configured")
	}

	return New(apiKey).(*client), nil
}

func (c *client) Members(ctx context.Context, orgID string) ([]Member, error) {
	memberships, err := c.directoryMemberships(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, nil
	}
	members := make(map[string]Member)
	seenCursors := make(map[string]struct{})
	after := ""
	userCount := 0
	for range maxDirectoryPages {
		page, err := c.listOrganizationUsers(ctx, orgID, after)
		if err != nil {
			return nil, errors.New("list workos organization users")
		}
		userCount += len(page.Data)
		if userCount > maxDirectoryMembers {
			return nil, fmt.Errorf("%w: users", ErrDirectoryLimit)
		}
		for _, user := range page.Data {
			membership, found := memberships[user.ID]
			if !found {
				continue
			}
			if member, valid := verifiedMember(membership, user); valid {
				members[member.UserID] = member
			}
		}
		if page.ListMetadata.After == "" {
			result := make([]Member, 0, len(members))
			for _, member := range members {
				result = append(result, member)
			}
			sort.Slice(result, func(i, j int) bool { return result[i].UserID < result[j].UserID })
			return result, nil
		}
		after = page.ListMetadata.After
		if _, seen := seenCursors[after]; seen {
			return nil, fmt.Errorf("%w: repeated user cursor", ErrDirectoryLimit)
		}
		seenCursors[after] = struct{}{}
	}
	return nil, fmt.Errorf("%w: user pages", ErrDirectoryLimit)
}

func (c *client) LookupMembers(ctx context.Context, orgID string, userIDs []string) ([]Member, error) {
	if len(userIDs) > maxDirectoryMembers {
		return nil, fmt.Errorf("%w: requested users", ErrDirectoryLimit)
	}
	members := make([]Member, 0, len(userIDs))
	seen := make(map[string]struct{})
	for _, userID := range userIDs {
		if _, found := seen[userID]; found || userID == "" {
			continue
		}
		seen[userID] = struct{}{}
		member, found, err := c.lookupMember(ctx, orgID, userID)
		if err != nil {
			return nil, err
		}
		if found {
			members = append(members, member)
		}
	}
	return members, nil
}

func (c *client) CanEmail(ctx context.Context, orgID, userID, email string) (bool, error) {
	member, found, err := c.lookupMember(ctx, orgID, userID)
	return found && member.Email == email, err
}

func (c *client) directoryMemberships(ctx context.Context, orgID string) (map[string]membership, error) {
	memberships := make(map[string]membership)
	seenCursors := make(map[string]struct{})
	after := ""
	membershipCount := 0

	for pageNumber := 0; pageNumber < maxDirectoryPages; pageNumber++ {
		page, err := c.listOrganizationMemberships(ctx, orgID, after)
		if err != nil {
			return nil, errors.New("list workos organization memberships")
		}

		membershipCount += len(page.Data)
		if membershipCount > maxDirectoryMembers {
			return nil, fmt.Errorf("%w: memberships", ErrDirectoryLimit)
		}

		for _, membership := range page.Data {
			if membership.OrganizationID != orgID || membership.Status != "active" || membership.UserID == "" {
				continue
			}
			if _, ok := memberships[membership.UserID]; ok {
				continue
			}
			memberships[membership.UserID] = membership
		}

		next := page.ListMetadata.After
		if next == "" {
			return memberships, nil
		}
		if _, ok := seenCursors[next]; ok {
			return nil, fmt.Errorf("%w: repeated membership cursor", ErrDirectoryLimit)
		}
		seenCursors[next] = struct{}{}
		after = next
	}

	return nil, fmt.Errorf("%w: membership pages", ErrDirectoryLimit)
}

func (c *client) lookupMember(ctx context.Context, orgID, userID string) (Member, bool, error) {
	query := url.Values{"organization_id": {orgID}, "user_id": {userID}, "statuses": {"active"}, "limit": {"100"}}
	var page membershipsPage
	if err := c.get(ctx, "/user_management/organization_memberships", query, &page); err != nil {
		return Member{}, false, errors.New("check workos organization membership")
	}
	for _, membership := range page.Data {
		if membership.OrganizationID != orgID || membership.UserID != userID || membership.Status != "active" {
			continue
		}
		user, err := c.getUser(ctx, userID)
		if err != nil {
			return Member{}, false, errors.New("get workos organization user")
		}
		member, valid := verifiedMember(membership, user)
		return member, valid, nil
	}
	return Member{UserID: "", Email: "", Admin: false}, false, nil
}

func verifiedMember(membership membership, user user) (Member, bool) {
	email := strings.TrimSpace(user.Email)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || !user.EmailVerified {
		return Member{UserID: "", Email: "", Admin: false}, false
	}
	return Member{UserID: membership.UserID, Email: email, Admin: membership.Role.Slug == adminRoleSlug}, true
}
