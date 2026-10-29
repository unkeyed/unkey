package grant

import "github.com/unkeyed/unkey/pkg/auth/principal"

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

// Authorization is the non-secret state of a grant request, for the consent
// page.
type Authorization struct {
	UserCode           string
	DeviceName         string
	RequesterIP        string
	RequesterUserAgent string
	Status             string
	WorkspaceID        string
	CreatedAt          int64
	ExpiresAt          int64
	// Client is the OAuth client asking for access. Nil for a CLI device
	// login.
	Client *ConsentClient
}

// ConsentClient is the OAuth client asking for access.
type ConsentClient struct {
	ID ClientID
	// Name was chosen by whoever registered the client. Do not trust it.
	Name string
	// RedirectURI is where approval sends the browser. Its host is the
	// trustworthy part of the consent screen.
	RedirectURI string
}

// Decided is where the approver's browser goes after a decision. RedirectTo
// is set only for OAuth requests: the client's redirect URI with code, state,
// and iss after approval, or error=access_denied after denial. For a CLI
// device login it is empty because the CLI collects the grant by polling.
type Decided struct {
	RedirectTo string
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

// RefreshRequest exchanges a refresh token for a new pair.
type RefreshRequest struct {
	RefreshToken string
	// Holder presents the token: CLI from cli.exchangeToken, HeldBy(id) from
	// the OAuth token endpoint. A token held by someone else is rejected
	// without rotating.
	Holder    Holder
	RemoteIP  string
	UserAgent string
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
	ID          string
	UserID      string
	DeviceName  string
	Permissions []string
	CreatedAt   int64
	LastUsedAt  int64
	ExpiresAt   int64
	RevokedAt   int64
	// ClientID is the OAuth client holding the session. Empty for the CLI.
	ClientID ClientID
}
