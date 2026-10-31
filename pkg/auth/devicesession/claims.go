// Package devicesession authenticates Unkey-signed user device session tokens.
//
// A user approves a device session on the dashboard for the Unkey CLI or
// another client. Its access token is an HS256 JWT whose permissions claim is
// the set the user picked at approval, already checked against their role. The
// API does not expand roles for these tokens. A short cache in front of the
// session row rejects revoked and expired sessions without a database read on
// every request.
package devicesession

import "github.com/unkeyed/unkey/pkg/jwt"

const (
	// TokenUse marks a JWT as a device session access token. Resolvers ignore
	// bearers that do not carry it, so dashboard and WorkOS tokens fall through.
	TokenUse = "device_session"

	// Issuer is the iss claim Unkey puts on device session access tokens.
	Issuer = "unkey-device-session"

	// Audience is the aud claim Unkey puts on device session access tokens.
	Audience = "unkey-api"
)

// Claims is the payload of a device session access token.
type Claims struct {
	jwt.RegisteredClaims

	WorkspaceID string   `json:"workspace_id"`
	SessionID   string   `json:"session_id"`
	Permissions []string `json:"permissions"`
	TokenUse    string   `json:"token_use"`
	Name        string   `json:"name,omitempty"`
	Device      string   `json:"device,omitempty"`
}
