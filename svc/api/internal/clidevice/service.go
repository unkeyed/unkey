package clidevice

import (
	"context"
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

	ratelimitScope      = "cli_device_login"
	startLimitPerMinute = 20
	pollLimitPerMinute  = 120
)

// PollStatus is the value cli.pollDeviceLogin returns in status.
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

// Service runs device login and CLI sessions.
type Service struct {
	DB               db.Database
	Auditlogs        auditlogs.AuditLogService
	Clock            clock.Clock
	Ratelimit        ratelimit.Service
	Memberships      Memberships
	DashboardBaseURL string
	Signer           tokenjwt.Signer[clisession.Claims]
	SessionCache     cache.Cache[string, SessionVerdict]
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

// RequireDashboardSession rejects root keys and CLI access tokens.
// Get, approve, and deny are dashboard actions.
func RequireDashboardSession(p *principal.Principal) error {
	if p == nil {
		return dashboardSessionRequired()
	}
	source, ok := p.Source.(principal.JWTSource)
	if !ok || len(source.Roles) == 0 || p.Subject.Type != principal.SubjectTypeUser || p.Subject.ID == "" {
		return dashboardSessionRequired()
	}
	return nil
}

// RequireUserSession accepts a dashboard session or a CLI access token.
// List and revoke use it. Root keys are rejected.
func RequireUserSession(p *principal.Principal) error {
	if err := RequireDashboardSession(p); err == nil {
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

func (s *Service) limit(ctx context.Context, action string, remoteIP string, perMinute int64) error {
	if s.Ratelimit == nil {
		return nil
	}
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
		Limit:       perMinute,
		Duration:    time.Minute,
		Cost:        1,
		Time:        time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	})
	if err != nil {
		logger.Error("cli device login rate limit: ratelimiter error", "action", action, "error", err.Error())
		return nil
	}
	if !resp.Success {
		return fault.New("cli device login rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.IPRateLimited.URN()),
			fault.Public("Too many CLI login requests from this address. Try again in a minute."),
		)
	}
	return nil
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
