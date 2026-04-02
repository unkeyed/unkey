package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/unkeyed/unkey/internal/services/sessionauth"
	"github.com/unkeyed/unkey/pkg/zen"
)

// WithSessionAuth returns a middleware that attempts JWT session authentication.
// If the Bearer token is a JWT and verifies successfully, it sets WorkspaceID
// and marks the session as session-authenticated. If the token is not a JWT or
// verification fails, the middleware does nothing and lets the request continue
// so the handler can try root key authentication instead.
//
// This allows every route to accept both root keys and session tokens without
// any handler changes.
func WithSessionAuth(svc sessionauth.Service) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			if svc == nil {
				return next(ctx, s)
			}

			token, err := zen.Bearer(s)
			if err != nil {
				// No bearer token — let handler deal with it.
				return next(ctx, s)
			}

			if !looksLikeJWT(token) {
				// Not a JWT — continue to handler for root key auth.
				return next(ctx, s)
			}

			result, err := svc.Authenticate(ctx, token)
			if err != nil {
				// JWT verification failed — continue to handler for root key auth.
				// This covers cases where a root key happens to look like a JWT.
				return next(ctx, s)
			}

			s.WorkspaceID = result.WorkspaceID
			s.SetSessionAuth(result.UserID)
			return next(ctx, s)
		}
	}
}

// looksLikeJWT checks if a token has the structure of a JWT (three
// base64url-encoded parts where the header contains an "alg" field).
// This is a fast heuristic to avoid sending root keys to the JWKS verifier.
func looksLikeJWT(token string) bool {
	parts := strings.SplitN(token, ".", 4)
	if len(parts) != 3 {
		return false
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return false
	}

	return header.Alg != ""
}
