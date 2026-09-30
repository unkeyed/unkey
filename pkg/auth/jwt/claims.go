package jwt

import (
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
)

// Audience is the default JWT audience for dashboard-minted API bearer JWTs.
const Audience = "api.unkey.com"

// Claims is the JWT payload accepted by the API auth resolver.
type Claims struct {
	tokenjwt.RegisteredClaims

	// Org scopes JWT principals to one organization.
	Org OrganizationClaims `json:"org"`

	// User supports providers that put user identity in a nested object.
	User UserClaims `json:"user"`

	// Name is optional display text for audit logs. Subject is used when empty.
	Name string `json:"name"`

	// Roles contains the assigned organization roles.
	Roles []string `json:"roles"`
}

// OrganizationClaims contains the organization identifier from a JWT.
type OrganizationClaims struct {
	ID string `json:"id"`
}

// UserClaims contains the user identifier from a JWT.
type UserClaims struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// subjectID returns the token subject, falling back to the nested user id used
// by providers that omit the standard sub claim.
func (c Claims) subjectID() string {
	if c.Subject != "" {
		return c.Subject
	}
	return c.User.ID
}

// subjectName returns display text for audit logs, falling back from the name
// claim to the user email to the subject id.
func (c Claims) subjectName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.User.Email != "" {
		return c.User.Email
	}
	return c.subjectID()
}
