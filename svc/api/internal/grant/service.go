package grant

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/logger"
)

const (
	pollIntervalSeconds = 5
	deviceTTL           = 10 * time.Minute
	accessTTL           = 15 * time.Minute
	sessionIdle         = 7 * 24 * time.Hour
	sessionMax          = 30 * 24 * time.Hour

	// refreshReuseGrace is how long the immediately previous refresh token
	// still rotates the session. A client whose response was lost can retry
	// inside this window without revoking the family. After it, that token
	// is reuse and the family is revoked.
	refreshReuseGrace = 10 * time.Second
	retainExpiredFor  = 24 * time.Hour

	// consentTTL is how long an OAuth authorization request waits for the
	// user to sign in and decide.
	consentTTL = 10 * time.Minute
	// codeTTL is the authorization code lifetime. Approval shrinks an OAuth
	// request's expiry to approval time plus codeTTL.
	codeTTL = 60 * time.Second
	// consentHandleLength is the user code length for OAuth requests. Nobody
	// types it, so it is long enough that guessing one is hopeless.
	consentHandleLength = 26

	ratelimitScope          = "cli_device_login"
	startLimitPerMinute     = 20
	pollLimitPerMinute      = 120
	userCodeLimit           = 60
	userCodeLimitWindow     = 10 * time.Minute
	authorizeLimitPerMinute = 30

	// Failed redemptions and refreshes are limited per address and only
	// failures count. The OAuth gateway sends every user's requests from a
	// few addresses, so a limit on all attempts, or one keyed on the public
	// client_id, would let anyone lock every user out.
	failedRedeemLimitPerMinute  = 30
	failedRefreshLimitPerMinute = 30
)

type PollStatus string

const (
	PollPermissionsRequired PollStatus = "permissions_required"
	PollDenied              PollStatus = "access_denied"
	PollExpired             PollStatus = "expired_token"
	PollComplete            PollStatus = "complete"
)

// Memberships resolves a user's current roles in a WorkOS organization.
// found is false when the user has no active membership. err is an
// infrastructure failure, not a missing membership.
type Memberships interface {
	ActiveRoles(ctx context.Context, orgID string, userID string) (roles []string, found bool, err error)
}

// Service runs grant requests, OAuth clients, and the sessions they issue.
type Service struct {
	DB               db.Database
	Auditlogs        auditlogs.AuditLogService
	Clock            clock.Clock
	Ratelimit        ratelimit.Service
	Memberships      Memberships
	DashboardBaseURL string
	Signer           tokenjwt.Signer[clisession.Claims]
	SessionCache     cache.Cache[string, SessionVerdict]

	// Issuer is the OAuth authorization server's issuer, the API's public
	// origin. It is the RFC 8414 issuer and the RFC 9207 iss parameter.
	// Empty disables OAuth; device login keeps working.
	Issuer string
	// RedirectOrigins, when non-empty, is the closed set of origins OAuth
	// clients may register redirect URIs under.
	RedirectOrigins []string
}

// SessionVerdict is the cached answer for one CLI session.
type SessionVerdict struct {
	Allowed bool
	Session clisession.Session
}

// Enabled reports whether login can issue tokens.
func (s *Service) Enabled() bool {
	return s != nil &&
		s.Signer != nil &&
		s.Memberships != nil &&
		strings.TrimSpace(s.DashboardBaseURL) != ""
}

func (s *Service) requireEnabled() error {
	if !s.Enabled() {
		return fault.New("cli auth is not configured",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public("CLI login is not configured on this API."),
		)
	}
	return nil
}

// OAuthEnabled reports whether the OAuth endpoints can run: everything device
// login needs plus an issuer.
func (s *Service) OAuthEnabled() bool {
	return s.Enabled() && strings.TrimSpace(s.Issuer) != ""
}

func (s *Service) requireOAuth() error {
	if !s.OAuthEnabled() {
		return fault.New("oauth is not configured",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public("OAuth is not configured on this API."),
		)
	}
	return nil
}

// RequireDashboardSession returns the dashboard JWT source and rejects other credentials.
func RequireDashboardSession(p *principal.Principal) (principal.JWTSource, error) {
	if p == nil {
		return principal.JWTSource{}, dashboardSessionRequired()
	}
	source, ok := p.Source.(principal.JWTSource)
	if !ok || len(source.Roles) == 0 || p.Subject.Type != principal.SubjectTypeUser || p.Subject.ID == "" {
		return principal.JWTSource{}, dashboardSessionRequired()
	}
	return source, nil
}

// RequireUserSession accepts a dashboard session or a CLI access token.
// List and revoke use it. Root keys are rejected.
func RequireUserSession(p *principal.Principal) error {
	if _, err := RequireDashboardSession(p); err == nil {
		return nil
	}
	if p == nil {
		return dashboardSessionRequired()
	}
	if _, ok := p.Source.(principal.CLISessionSource); ok && p.Subject.Type == principal.SubjectTypeUser && p.Subject.ID != "" {
		return nil
	}
	return dashboardSessionRequired()
}

func dashboardSessionRequired() error {
	return fault.New("cli login requires a dashboard session",
		fault.Code(codes.Auth.Authorization.Forbidden.URN()),
		fault.Public("Sign in to the dashboard to manage this login."),
	)
}

func workspaceUsable(workspace db.Workspace) bool {
	return !workspace.DeletedAtM.Valid && workspace.Enabled && strings.TrimSpace(workspace.OrgID) != ""
}

func (s *Service) limitUserCodeAttempts(ctx context.Context, userID string) error {
	resp, err := s.Ratelimit.Ratelimit(ctx, ratelimit.RatelimitRequest{
		WorkspaceID: ratelimitScope,
		Namespace:   ratelimitScope + ".user_code",
		Identifier:  userID,
		Limit:       userCodeLimit,
		Duration:    userCodeLimitWindow,
		Cost:        1,
		Time:        s.Clock.Now(),
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("cli user code rate limit failed"),
			fault.Public("Login is temporarily unavailable. Try again later."),
		)
	}
	if !resp.Success {
		return fault.New("cli user code rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.UserRateLimited.URN()),
			fault.Public("Too many login attempts. Wait 10 minutes and try again."),
		)
	}
	return nil
}

func (s *Service) limit(ctx context.Context, action string, remoteIP string, perMinute int64) error {
	return s.limitAddress(ctx, action, remoteIP, perMinute, time.Minute, "Too many CLI login requests from this address. Try again in a minute.")
}

func (s *Service) limitAddress(ctx context.Context, action string, remoteIP string, limit int64, window time.Duration, message string) error {
	// The identifier is the client IP. Start and poll run before any workspace
	// is chosen, and the request does not carry one. A workspace id would be
	// missing or caller-supplied, so it cannot key this limit. Per-IP keeps
	// the unauthenticated endpoints from being driven by one address.
	identifier := remoteIP
	if identifier == "" {
		identifier = "unknown"
	}
	resp, err := s.Ratelimit.Ratelimit(ctx, ratelimit.RatelimitRequest{
		WorkspaceID: ratelimitScope,
		Namespace:   ratelimitScope + "." + action,
		Identifier:  identifier,
		Limit:       limit,
		Duration:    window,
		Cost:        1,
		Time:        time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	})
	if err != nil {
		logger.Error("grant rate limit: ratelimiter error", "action", action, "error", err.Error())
		return nil
	}
	if !resp.Success {
		return fault.New("grant rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.IPRateLimited.URN()),
			fault.Public(message),
		)
	}
	return nil
}

// limitFailures runs attempt unless remoteIP already failed limit times in
// the last minute, and counts the attempt when it is refused with
// ErrRejected. The limiter failing open matches limitAddress.
func (s *Service) limitFailures(ctx context.Context, action string, remoteIP string, limit int64, attempt func() (Tokens, error)) (Tokens, error) {
	identifier := remoteIP
	if identifier == "" {
		identifier = "unknown"
	}
	failures := ratelimit.RatelimitRequest{
		WorkspaceID: ratelimitScope,
		Namespace:   ratelimitScope + "." + action,
		Identifier:  identifier,
		Limit:       limit,
		Duration:    time.Minute,
		Cost:        0,
		Time:        time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	}
	peek, err := s.Ratelimit.Ratelimit(ctx, failures)
	if err != nil {
		logger.Error("grant failure limit: ratelimiter error", "action", action, "error", err.Error())
	} else if !peek.Success || peek.Remaining == 0 {
		return Tokens{}, fault.New("grant failure limit exceeded",
			fault.Code(codes.User.TooManyRequests.IPRateLimited.URN()),
			fault.Public("Too many failed attempts from this address. Try again in a minute."),
		)
	}

	tokens, err := attempt()
	if errors.Is(err, ErrRejected) {
		failures.Cost = 1
		if _, countErr := s.Ratelimit.Ratelimit(ctx, failures); countErr != nil {
			logger.Error("grant failure limit: ratelimiter error", "action", action, "error", countErr.Error())
		}
	}
	return tokens, err
}

// sessionExpiry is the idle deadline for a session used at now. It never
// passes the absolute maximum measured from createdAt.
func sessionExpiry(createdAt int64, now int64) int64 {
	return min(now+sessionIdle.Milliseconds(), createdAt+sessionMax.Milliseconds())
}

func (s *Service) deleteExpired(ctx context.Context) {
	cutoff := s.Clock.Now().Add(-retainExpiredFor).UnixMilli()
	if _, err := db.Query.DeleteExpiredDeviceAuthorizations(ctx, s.DB.RW(), cutoff); err != nil {
		logger.Error("device authorization cleanup failed", "error", err.Error())
	}
	if _, err := db.Query.DeleteExpiredCLISessions(ctx, s.DB.RW(), cutoff); err != nil {
		logger.Error("cli session cleanup failed", "error", err.Error())
	}
}
