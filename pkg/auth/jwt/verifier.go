// Package jwt verifies short-lived JWTs minted by trusted internal callers
// (today: the dashboard's /proxy route) and exposes the authenticated
// subject as an auth.Principal so handlers stay agnostic to whether the
// caller used a root key or a JWT.
//
// The verifier is deliberately strict because svc/api is internet-facing:
// any caller can submit an arbitrary "JWT" and we have to assume it's
// hostile. We enforce HS256-only, require iss/aud/exp/iat/nbf/sub claims,
// and cap the claimed lifetime so a forged iat can't widen the validity
// window.
package jwt

import (
	"errors"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/unkeyed/unkey/pkg/auth"
)

const (
	// Issuer is the only iss claim we accept. Tokens minted by anything
	// else are rejected, including older clients that omit iss entirely.
	// Must match the issuer the dashboard /proxy route sets when minting.
	// Use the public dashboard domain so the value is stable across
	// environments and self-documenting.
	Issuer = "app.unkey.com"

	// Audience is the only aud claim we accept. Defends against a token
	// minted for a different Unkey service being replayed against this
	// one. Must match the audience the dashboard /proxy route sets. Use
	// the public API domain so the value is stable across environments.
	Audience = "api.unkey.com"

	// MaxLifetime caps the claimed exp - iat span. The dashboard mints
	// 2-minute tokens; this 5-minute ceiling tolerates clock skew without
	// opening a real attack window. A token claiming a longer lifetime
	// is rejected even if iat, exp, and the signature are all valid.
	MaxLifetime = 5 * time.Minute

	// leeway is the clock-skew tolerance applied to exp/nbf/iat checks.
	// 30s is wide enough for inter-region drift on managed infra and
	// narrow enough that a stolen token can't be replayed long after exp.
	leeway = 30 * time.Second
)

// Claims is the JWT payload accepted by the API. It embeds the standard
// registered claims (iat/exp/nbf/iss/aud/sub) and adds the workspace and
// permission data the API needs to authorize requests.
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
// secret, enforces all required claims (iss, aud, exp, iat, nbf, sub,
// wid), and converts the result to an auth.Principal. On any failure it
// returns an error.
//
// The signing method is enforced to HS256 in two layers (gojwt's
// WithValidMethods plus an explicit type assertion in the keyfunc) to
// prevent algorithm-confusion attacks (e.g. alg=none, or an asymmetric
// public key being treated as an HMAC secret).
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
	},
		gojwt.WithValidMethods([]string{"HS256"}),
		gojwt.WithExpirationRequired(),
		gojwt.WithIssuedAt(),
		gojwt.WithIssuer(Issuer),
		gojwt.WithAudience(Audience),
		gojwt.WithLeeway(leeway),
		gojwt.WithStrictDecoding(),
	)
	if err != nil {
		return nil, fmt.Errorf("jwt: %w", err)
	}

	// gojwt does not enforce the presence of iat or nbf even when their
	// validators are enabled (only exp can be required-by-config). We
	// require them manually: missing iat means we can't compute the
	// lifetime cap, and missing nbf sidesteps the not-before guard.
	if claims.IssuedAt == nil {
		return nil, errors.New("jwt: missing iat claim")
	}
	if claims.NotBefore == nil {
		return nil, errors.New("jwt: missing nbf claim")
	}
	if claims.Subject == "" {
		return nil, errors.New("jwt: missing sub claim")
	}
	if claims.WorkspaceID == "" {
		return nil, errors.New("jwt: missing wid claim")
	}

	// Cap the claimed lifetime as defense in depth: even with a valid
	// signature and consistent iat/exp pair, a token claiming a multi-hour
	// lifetime is rejected. Bounds blast radius if iat is bogus.
	if d := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); d > MaxLifetime {
		return nil, fmt.Errorf("jwt: claimed lifetime %s exceeds max %s", d, MaxLifetime)
	}

	return &auth.Principal{
		Scheme:      auth.SchemeJWT,
		ID:          claims.Subject,
		DisplayName: claims.Name,
		WorkspaceID: claims.WorkspaceID,
		Permissions: claims.Permissions,
		Authorizer:  auth.GrantedPermissions(claims.Permissions),
	}, nil
}
