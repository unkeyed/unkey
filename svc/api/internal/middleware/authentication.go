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
	principalauth "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
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

	// KeyVerifications records root key authentication and authorization outcomes.
	KeyVerifications *batch.BatchProcessor[schema.KeyVerification]

	// Region identifies the API region that authenticated the root key.
	Region string

	// Database loads workspace limit rows when they are not cached.
	Database db.Database

	// LimitsCache caches workspace limit rows by workspace ID.
	LimitsCache cache.Cache[string, keysdb.Limit]

	// Ratelimit enforces workspace-level API request limits.
	Ratelimit ratelimit.Service
}

// WithAuthentication authenticates the request and applies workspace-level rate limiting.
//
// Handlers behind this middleware can call [zen.Session.GetPrincipal] and then
// perform route-specific authorization. Workspace rate limiting lives here so
// every credential source is checked consistently after authentication resolves
// the workspace and before business logic runs. Root key usage is recorded at
// this boundary because later route authorization can have multiple outcomes.
func WithAuthentication(config AuthenticationConfig) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, sess *zen.Session) error {
			startedAt := time.Now()
			p, err := config.Auth.Authenticate(ctx, sess)
			if err != nil {
				return err
			}

			if p.Subject.Type == principalauth.SubjectTypeRootKey && config.KeyVerifications != nil {
				keySource, _ := p.Source.(principalauth.KeySource)

				verification := schema.KeyVerification{
					RequestID:    sess.RequestID(),
					Time:         time.Now().UnixMilli(),
					WorkspaceID:  p.WorkspaceID,
					KeySpaceID:   keySource.KeySpaceID,
					IdentityID:   "",
					ExternalID:   "",
					KeyID:        p.Subject.ID,
					Region:       config.Region,
					Source:       schema.SourceAPI,
					AppID:        "",
					Outcome:      schema.OutcomeValid,
					Tags:         []string{},
					SpentCredits: 0,
					Latency:      float64(time.Since(startedAt).Milliseconds()),
				}

				defer func() {
					if principalauth.AuthorizationError(p) != nil {
						verification.Outcome = schema.OutcomeInsufficientPermissions
					}
					config.KeyVerifications.Buffer(verification)
				}()
			}

			if err := checkWorkspaceRateLimit(ctx, sess, config, p.WorkspaceID); err != nil {
				return err
			}

			return next(ctx, sess)
		}
	}
}

func checkWorkspaceRateLimit(ctx context.Context, sess *zen.Session, config AuthenticationConfig, workspaceID string) error {
	if config.LimitsCache == nil || config.Ratelimit == nil {
		return nil
	}

	limits, _, err := config.LimitsCache.SWR(ctx, workspaceID, func(ctx context.Context) (keysdb.Limit, error) {
		return keysdb.Query.FindLimitsByWorkspaceID(ctx, config.Database.RO(), workspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		logger.Error("workspace rate limit: failed to load limits",
			"workspace_id", workspaceID,
			"error", err.Error(),
		)
		// Workspace API rate limiting fails open when limit lookup is unavailable.
		return nil
	}

	if !limits.ApiRequestsCountMaxPerMinute.Valid {
		return nil
	}

	limit := limits.ApiRequestsCountMaxPerMinute.Int32
	duration := time.Minute

	if limit == 0 {
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
