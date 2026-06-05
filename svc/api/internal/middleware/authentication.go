package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

const workspaceRatelimitNamespace = "workspace.ratelimit"

// AuthenticationConfig configures authentication and workspace-level request policy.
type AuthenticationConfig struct {
	// Auth resolves request credentials into a session principal.
	Auth auth.Authenticator

	// Database loads workspace quota rows when they are not cached.
	Database db.Database

	// QuotaCache caches workspace quota rows by workspace ID.
	QuotaCache cache.Cache[string, keysdb.Quotas]

	// Ratelimit enforces workspace-level API request quotas.
	Ratelimit ratelimit.Service
}

// WithAuthentication authenticates the request and applies workspace-level rate limiting.
//
// Handlers behind this middleware can call [zen.Session.GetPrincipal] and then
// perform route-specific authorization. Workspace rate limiting lives here so
// every credential source is checked consistently after authentication resolves
// the workspace and before business logic runs.
func WithAuthentication(config AuthenticationConfig) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, sess *zen.Session) error {
			principal, err := config.Auth.Authenticate(ctx, sess)
			if err != nil {
				return err
			}

			if err := checkWorkspaceRateLimit(ctx, sess, config, principal.WorkspaceID); err != nil {
				return err
			}

			return next(ctx, sess)
		}
	}
}

func checkWorkspaceRateLimit(ctx context.Context, sess *zen.Session, config AuthenticationConfig, workspaceID string) error {
	if config.QuotaCache == nil || config.Ratelimit == nil {
		return nil
	}

	quota, _, err := config.QuotaCache.SWR(ctx, workspaceID, func(ctx context.Context) (keysdb.Quotas, error) {
		return keysdb.Query.FindQuotaByWorkspaceID(ctx, config.Database.RO(), workspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		logger.Error("workspace rate limit: failed to load quota",
			"workspace_id", workspaceID,
			"error", err.Error(),
		)
		// Workspace API rate limiting fails open when quota lookup is unavailable.
		return nil
	}

	if !quota.RatelimitApiLimit.Valid || !quota.RatelimitApiDuration.Valid {
		return nil
	}

	limit := quota.RatelimitApiLimit.Int32
	duration := time.Duration(quota.RatelimitApiDuration.Int32) * time.Millisecond

	if limit == 0 || duration == 0 {
		return fault.New("workspace rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.WorkspaceRateLimited.URN()),
			fault.Internal("workspace rate limit is zero"),
			fault.Public(
				fmt.Sprintf("This workspace has exceeded its API rate limit of %d/%s. Please try again later.", limit, duration.String()),
			),
		)
	}

	resp, err := config.Ratelimit.Ratelimit(ctx, ratelimit.RatelimitRequest{
		WorkspaceID: workspaceID,
		Namespace:   workspaceRatelimitNamespace,
		Identifier:  workspaceID,
		Limit:       int64(limit),
		Duration:    duration,
		Cost:        1,
		Time:        time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	})
	if err != nil {
		logger.Error("workspace rate limit: ratelimiter error",
			"workspace_id", workspaceID,
			"error", err.Error(),
		)
		// Workspace API rate limiting fails open when the limiter backend is unavailable.
		return nil
	}

	resetSeconds := max(int64(time.Until(resp.Reset).Seconds()), 0)

	sess.AddHeader("RateLimit-Limit", strconv.FormatInt(resp.Limit, 10))
	sess.AddHeader("RateLimit-Remaining", strconv.FormatInt(resp.Remaining, 10))
	sess.AddHeader("RateLimit-Reset", strconv.FormatInt(resetSeconds, 10))

	if !resp.Success {
		sess.AddHeader("Retry-After", strconv.FormatInt(resetSeconds, 10))

		return fault.New("workspace rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.WorkspaceRateLimited.URN()),
			fault.Internal("workspace rate limit exceeded"),
			fault.Public(
				fmt.Sprintf("This workspace has exceeded its API rate limit of %d/%s. Please try again later.", limit, duration.String()),
			),
		)
	}

	return nil
}
