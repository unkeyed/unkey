package middleware

import (
	"context"

	"github.com/unkeyed/unkey/internal/services/sessionauth"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

// WithSessionAuth returns a middleware that attempts session authentication.
// If the service can handle the token and authentication succeeds, it sets
// WorkspaceID and marks the session as session-authenticated. Otherwise the
// middleware does nothing and lets the handler try root key auth instead.
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

			if !svc.CanHandle(token) {
				// Service doesn't recognize this token format — continue
				// to handler for root key auth.
				return next(ctx, s)
			}

			result, err := svc.Authenticate(ctx, token)
			if err != nil {
				// Authentication failed — continue to handler for root key auth.
				logger.Error("session auth failed, falling back to root key",
					"error", err.Error(),
				)
				return next(ctx, s)
			}

			s.WorkspaceID = result.WorkspaceID
			s.SetSessionAuth(result.UserID)
			return next(ctx, s)
		}
	}
}
