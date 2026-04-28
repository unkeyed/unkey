// Package jwt verifies short-lived JWTs minted by trusted internal callers
// (today: the dashboard's /proxy route) and exposes the authenticated
// subject as an auth.Principal so handlers stay agnostic to whether the
// caller used a root key or a JWT.
//
// The verifier is intentionally minimal. It does not handle issuance;
// signing happens wherever the JWT is minted (the dashboard proxy in our
// first use case). Anything authoritative (workspace scoping, granted
// permissions) is read from claims and trusted only because the signature
// verified.
package jwt

import (
	"errors"
	"fmt"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/unkeyed/unkey/pkg/auth"
)

// Claims is the JWT payload accepted by the API. It embeds the standard
// registered claims (iat/exp/sub/etc.) and adds the workspace + permission
// data the API needs to authorize requests.
type Claims struct {
	gojwt.RegisteredClaims

	// WorkspaceID identifies the tenant the bearer is authorized to act on.
	WorkspaceID string `json:"wid"`

	// Name is a human-readable label for audit logs (user email, "dashboard").
	Name string `json:"name"`

	// Permissions are granted permission strings in "resource.id.action" form.
	Permissions []string `json:"perms"`
}

// Verify parses tokenString, validates the HMAC-SHA256 signature against
// secret, checks standard time claims (exp, nbf), and converts the result
// to an auth.Principal. On any failure it returns an error.
//
// The signing method is enforced to HS256 to prevent algorithm-confusion
// attacks (e.g. a token signed with "none" or with an attacker-controlled
// asymmetric key being accepted because the verifier wasn't strict).
func Verify(tokenString string, secret []byte) (*auth.Principal, error) {
	if len(secret) == 0 {
		return nil, errors.New("jwt: empty signing secret")
	}

	// nolint:exhaustruct // claims are populated by ParseWithClaims below.
	claims := &Claims{}
	_, err := gojwt.ParseWithClaims(tokenString, claims, func(token *gojwt.Token) (any, error) {
		if _, ok := token.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	}, gojwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, fmt.Errorf("jwt: %w", err)
	}

	if claims.WorkspaceID == "" {
		return nil, errors.New("jwt: missing wid claim")
	}
	// Require an explicit exp. golang-jwt v5 only validates exp when the
	// claim is present; a token without exp would never expire, which
	// defeats the point of using JWTs as short-lived bearers.
	if claims.ExpiresAt == nil {
		return nil, errors.New("jwt: missing exp claim")
	}

	return &auth.Principal{
		Scheme:      auth.SchemeJWT,
		ID:          claims.Subject,
		DisplayName: claims.Name,
		WorkspaceID: claims.WorkspaceID,
		Permissions: claims.Permissions,
	}, nil
}
