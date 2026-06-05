package principal

import (
	"github.com/unkeyed/unkey/pkg/rbac"
)

// Version is the current schema version for API auth principals.
const Version = "v1"

// Type identifies the authentication method that produced a principal.
type Type string

const (
	// TypeAPIKey is emitted when an API key authenticated the request.
	TypeAPIKey Type = "API_KEY"

	// TypeJWT is emitted when a JWT authenticated the request.
	TypeJWT Type = "JWT"

	// TypePortalSession is emitted when a portal session authenticated the request.
	TypePortalSession Type = "PORTAL_SESSION"
)

// SubjectType classifies the entity represented by a principal subject.
type SubjectType string

const (
	// SubjectTypeRootKey indicates the request was authenticated by a root key.
	SubjectTypeRootKey SubjectType = "rootkey"

	// SubjectTypeUser indicates the request was authenticated as an end user.
	SubjectTypeUser SubjectType = "user"
)

// Principal is the normalized authenticated subject used by API handlers.
//
// The envelope intentionally mirrors the frontline principal shape while this
// package stays independent from svc/frontline/internal packages.
type Principal struct {
	// Version is the schema version of the principal payload.
	Version string

	// Subject carries the audit-relevant identity of the authenticated entity.
	Subject Subject

	// Type identifies which authentication method produced this principal.
	Type Type

	// Source carries the method-specific authentication details.
	Source Source

	// WorkspaceID scopes all handler reads and writes for this principal.
	WorkspaceID string

	// Permissions is the flat RBAC permission set granted to this principal.
	Permissions []string
}

// Authorize evaluates this principal's permissions against the query.
func (p *Principal) Authorize(query rbac.PermissionQuery) error {
	return rbac.Check(query, p.Permissions)
}

// Subject identifies the authenticated entity and how it appears in audit logs.
type Subject struct {
	// ID is the stable identifier of the authenticated entity.
	ID string

	// Name is the human-readable subject name used for audit logs.
	Name string

	// Type classifies the subject for downstream audit logging.
	Type SubjectType
}

// Source is the discriminated union over authentication-method details.
type Source interface {
	principalSource()
}

// KeySource carries the API key detail that authenticated the request.
type KeySource struct {
	// KeyID is the ID of the key that authenticated the request.
	KeyID string

	// KeySpaceID is the key space that owns the authenticated key.
	KeySpaceID string

	// Permissions are the raw RBAC permission strings attached to the key.
	Permissions []string
}

func (KeySource) principalSource() {}

// JWTSource carries decoded JWT details used by the API auth resolver.
//
// This shape is a WIP placeholder for JWT authentication. The API does not
// register JWT authentication yet.
type JWTSource struct {
	// Header is the decoded token header, when captured by the resolver.
	Header map[string]any

	// Payload is the decoded token payload with claims preserved by name.
	Payload map[string]any

	// Signature is the raw signature string from the token's third segment.
	Signature string
}

func (JWTSource) principalSource() {}

// PortalSessionSource carries the portal session detail that authenticated the request.
//
// This shape is a WIP placeholder for portal auth. It keeps the permissions
// granted by the portal session until the final portal principal contract lands.
type PortalSessionSource struct {
	// SessionID is the portal browser session token ID.
	SessionID string

	// PortalConfigID is the portal configuration that issued the session.
	PortalConfigID string

	// ExternalID is the caller-assigned end-user identifier for the portal session.
	ExternalID string

	// Permissions are the raw RBAC permission strings attached to the portal session.
	Permissions []string
}

func (PortalSessionSource) principalSource() {}
