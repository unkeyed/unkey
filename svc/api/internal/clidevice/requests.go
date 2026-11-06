package clidevice

import (
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/db"
)

// StartRequest is an unauthenticated cli.startDeviceLogin call.
type StartRequest struct {
	DeviceName string
	RemoteIP   string
	UserAgent  string
}

// Started is the secret and the code the CLI shows.
type Started struct {
	LoginID         string
	UserCode        string
	VerificationURL string
	ExpiresIn       int32
	Interval        int32
}

// GetRequest loads one authorization for the approval page.
type GetRequest struct {
	Principal *principal.Principal
	UserCode  string
}

// Authorization is the non-secret state of a device login.
type Authorization struct {
	UserCode           string
	DeviceName         string
	RequesterIP        string
	RequesterUserAgent string
	Status             string
	WorkspaceID        string
	CreatedAt          int64
	ExpiresAt          int64
	// OAuth is set when an MCP client started the login through OAuth.
	OAuth *AuthorizationOAuth
}

// AuthorizationOAuth is what the consent page shows about an OAuth login.
type AuthorizationOAuth struct {
	ClientName string
	ClientHost string
	Resource   string
}

// ApproveRequest records the permissions the dashboard user granted.
type ApproveRequest struct {
	Principal   *principal.Principal
	UserCode    string
	Permissions []string
}

// DenyRequest rejects a login.
type DenyRequest struct {
	Principal *principal.Principal
	UserCode  string
}

type PollRequest struct {
	LoginID   string
	RemoteIP  string
	UserAgent string
}

// PollResult is the current login. Tokens are set only when Status is complete.
type PollResult struct {
	Status   PollStatus
	Interval int32
	Detail   string
	Tokens
}

// RefreshRequest exchanges a refresh token for a new pair. ClientID is the
// OAuth client presenting it, empty for the CLI, and must be the client the
// session was issued to. Resource, when set, must be the session's resource.
type RefreshRequest struct {
	RefreshToken string
	ClientID     string
	Resource     string
	RemoteIP     string
	UserAgent    string
}

// Tokens is a new access token and the next refresh token.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	WorkspaceID  string
	SessionID    string
}

// RevokeRequest revokes one session in the caller's workspace.
type RevokeRequest struct {
	Principal *principal.Principal
	SessionID string
	RemoteIP  string
	UserAgent string
}

// ListRequest lists sessions the caller is allowed to see.
type ListRequest struct {
	Principal *principal.Principal
}

// SessionInfo is one row in the sessions list. Token hashes are not included.
type SessionInfo struct {
	ID         string
	UserID     string
	Kind       db.DeviceSessionsKind
	DeviceName string
	// ClientHost is the host shown next to an mcp session's client name. It
	// is empty for CLI sessions and dynamically registered clients.
	ClientHost  string
	Resource    string
	Permissions []string
	CreatedAt   int64
	LastUsedAt  int64
	ExpiresAt   int64
	RevokedAt   int64
}
