// Package caller reads the caller's Unkey credential from an MCP request.
//
// svc/mcp never decides that a credential is valid: it holds no signing
// secret and no database, and the API verifies the credential on every
// forwarded tool call. This package only parses the credential and rejects
// session tokens it can see are unusable without verifying them. Only an HTTP
// 401 from the MCP endpoint makes an OAuth client refresh or sign in again,
// and a failed tool call cannot produce one.
package caller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/auth/clisession"
)

// Kind is how the caller authenticated. It is an analytics dimension.
type Kind string

const (
	// KindOAuth is an Unkey session access token issued through OAuth.
	KindOAuth Kind = "oauth"
	// KindRootKey is a credential that is not a JWT. The API decides
	// whether it is a valid root key.
	KindRootKey Kind = "root_key"
)

// rootKeyPrefix starts every root key the API issues.
const rootKeyPrefix = "unkey_"

// headerNames in precedence order. The Mcp-Unkey-V2-* headers are what the
// Speakeasy install pages told users to set; both carry a root key.
var headerNames = [...]string{"Authorization", "Mcp-Unkey-V2-Bearer", "Mcp-Unkey-V2-Root-Key"}

// expiryMargin rejects tokens about to expire, so the client refreshes before
// the API's own expiry check can fail the tool call.
const expiryMargin = 30 * time.Second

var (
	// ErrMissing means no header carried a credential.
	ErrMissing = errors.New("no credential")
	// ErrUnusable means the credential is visibly unusable: a JWT that is
	// not an Unkey session token, an Authorization bearer that is neither a
	// session token nor a root key, or a session token that is expiring or
	// was issued for another resource.
	ErrUnusable = errors.New("unusable credential")
)

// Credential is the caller's secret plus what svc/mcp can read from it
// without verifying it. Construct it with [Parse].
type Credential struct {
	secret string
	// claims are the unverified claims of an OAuth session token, nil for
	// root keys.
	claims *clisession.Claims
}

// Parse takes the credential from the first non-empty header in precedence
// order. Authorization must use the Bearer scheme; the legacy headers may
// omit it. A JWT whose token_use is the Unkey session token use is
// [KindOAuth] and any other JWT is [ErrUnusable]. Any other value is
// [KindRootKey], except that an Authorization bearer must also start with
// the root key prefix or it is [ErrUnusable].
func Parse(header http.Header) (Credential, error) {
	for _, name := range headerNames {
		value := strings.TrimSpace(header.Get(name))
		if value == "" {
			continue
		}
		scheme, token, hasScheme := strings.Cut(value, " ")
		switch {
		case hasScheme && strings.EqualFold(scheme, "bearer"):
			value = strings.TrimSpace(token)
		case name == "Authorization":
			return Credential{}, ErrUnusable
		}
		if value == "" {
			continue
		}
		credential, err := classify(value)
		// A client that signed in through Speakeasy's OAuth on the legacy URLs
		// still sends Speakeasy's token here. Only a 401 makes it sign in to
		// Unkey instead; forwarding the token would fail every tool call.
		if err == nil && name == "Authorization" && credential.claims == nil && !strings.HasPrefix(value, rootKeyPrefix) {
			return Credential{}, ErrUnusable
		}
		return credential, err
	}
	return Credential{}, ErrMissing
}

func classify(secret string) (Credential, error) {
	parts := strings.Split(secret, ".")
	if len(parts) != 3 {
		return Credential{secret: secret, claims: nil}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Credential{}, ErrUnusable
	}
	var claims clisession.Claims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.TokenUse != clisession.TokenUse {
		return Credential{}, ErrUnusable
	}
	return Credential{secret: secret, claims: &claims}, nil
}

// Kind reports how the caller authenticated.
func (c Credential) Kind() Kind {
	if c.claims == nil {
		return KindRootKey
	}
	return KindOAuth
}

// Authorization is the header value svc/mcp sends to the API.
func (c Credential) Authorization() string { return "Bearer " + c.secret }

// Triage returns [ErrUnusable] for a session token that expires within
// [expiryMargin] of now or has no audience on origin, the public origin of
// svc/mcp. CLI tokens carry only the API audience, so they cannot be replayed
// against svc/mcp. Root keys always pass. A nil error grants nothing: the API
// still verifies the credential.
func (c Credential) Triage(now time.Time, origin string) error {
	if c.claims == nil {
		return nil
	}
	if c.claims.ExpiresAt == 0 || !now.Add(expiryMargin).Before(time.Unix(c.claims.ExpiresAt, 0)) {
		return ErrUnusable
	}
	for _, audience := range c.claims.Audience {
		if Origin(audience) == origin {
			return nil
		}
	}
	return ErrUnusable
}

// SessionID is the session claimed by an OAuth token, unverified. Root-key
// calls return "".
func (c Credential) SessionID() string {
	if c.claims == nil {
		return ""
	}
	return c.claims.SessionID
}

// WorkspaceID is the workspace claimed by an OAuth token, returned only when
// the API answered 2xx to that token. The API validates request bodies before
// it authenticates, so a 4xx can come back for a forged token, while a 2xx
// proves the API verified it. Root-key calls return "".
func (c Credential) WorkspaceID(apiStatus int) string {
	if c.claims == nil || apiStatus < 200 || apiStatus > 299 {
		return ""
	}
	return c.claims.WorkspaceID
}

// String keeps the secret out of logs.
func (c Credential) String() string { return "credential(" + string(c.Kind()) + ")" }

// Origin returns scheme://host of an absolute URL in lower case, or "" when
// raw is not one.
func Origin(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
}
