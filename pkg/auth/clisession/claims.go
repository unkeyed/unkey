// Package clisession authenticates Unkey-signed CLI access tokens.
//
// A CLI session token is an HS256 JWT. Its permissions claim is the set the
// user picked at approval, already checked against their role. The API does
// not expand roles for these tokens. A short cache in front of the session
// row rejects revoked and expired sessions without a database read on every
// request.
package clisession

import (
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/jwt"
)

const (
	// TokenUse marks a JWT as a CLI access token. Resolvers ignore bearers
	// that do not carry it, so dashboard and WorkOS tokens fall through.
	TokenUse = "cli_session"

	// Issuer is the iss claim Unkey puts on CLI access tokens.
	Issuer = "unkey-cli"

	// Audience is the aud claim Unkey puts on CLI access tokens.
	Audience = "unkey-api"

	// AccessTokenLifetime is how long a CLI or MCP access token is valid.
	AccessTokenLifetime = 15 * time.Minute
)

// Claims is the payload of a CLI access token.
type Claims struct {
	jwt.RegisteredClaims

	WorkspaceID string   `json:"workspace_id"`
	SessionID   string   `json:"session_id"`
	Permissions []string `json:"permissions"`
	TokenUse    string   `json:"token_use"`
	Name        string   `json:"name,omitempty"`
	Device      string   `json:"device,omitempty"`
}

// issuedToMCPClient reports whether an MCP client holds the token. Tokens
// issued through OAuth carry the MCP server's resource in aud beside
// [Audience]; CLI tokens carry only [Audience].
func (c Claims) issuedToMCPClient() bool {
	return slices.ContainsFunc(c.Audience, func(aud string) bool { return aud != Audience })
}
