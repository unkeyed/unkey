package clidevice

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

const testSecret = "test-cli-signing-secret-32-bytes!"

type mem struct {
	roles map[string][]string
	gone  map[string]bool
}

func (m mem) ActiveRoles(_ context.Context, _ string, userID string) ([]string, bool, error) {
	if m.gone[userID] {
		return nil, false, nil
	}
	if roles, ok := m.roles[userID]; ok {
		return roles, true, nil
	}
	return []string{"admin"}, true, nil
}

func newService(h *testutil.Harness, membership mem) *Service {
	signer, err := tokenjwt.NewHS256Signer[clisession.Claims]([]byte(testSecret))
	if err != nil {
		panic(err)
	}
	if membership.roles == nil {
		membership.roles = map[string][]string{}
	}
	if membership.gone == nil {
		membership.gone = map[string]bool{}
	}
	return &Service{
		DB:               h.DB,
		Auditlogs:        h.Auditlogs,
		Clock:            h.Clock,
		Ratelimit:        h.Ratelimit,
		Memberships:      membership,
		DashboardBaseURL: "http://localhost:3000",
		Signer:           signer,
		SessionCache:     cache.NewNoopCache[string, SessionVerdict](),
	}
}

func adminPrincipal(workspace db.Workspace, userID string) *principal.Principal {
	return &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: userID, Name: "Admin", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions:           workos.PermissionsForRoles(workspace.ID, []string{"admin"}),
	}
}

func developerPrincipal(workspace db.Workspace, userID string) *principal.Principal {
	p := adminPrincipal(workspace, userID)
	p.Source = principal.JWTSource{Roles: []string{"developer"}}
	p.Permissions = workos.PermissionsForRoles(workspace.ID, []string{"developer"})
	return p
}

func permissionOnlyDeveloperHas(t *testing.T, workspaceID string) string {
	t.Helper()
	developer := workos.PermissionsForRoles(workspaceID, []string{"developer"})
	viewer := workos.PermissionsForRoles(workspaceID, []string{"viewer"})
	for _, permission := range developer {
		if !slices.Contains(viewer, permission) {
			return permission
		}
	}
	t.Fatal("developer role has no permission beyond viewer")
	return ""
}

func TestUserCodeAttemptsShareAUserBudgetAcrossActionsAndWorkspaces(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newService(h, mem{})
	limiter := ratelimit.NewLocal(h.Clock)
	t.Cleanup(func() { require.NoError(t, limiter.Close()) })
	svc.Ratelimit = limiter
	workspace := h.Resources().UserWorkspace
	p := adminPrincipal(workspace, uid.New(uid.TestPrefix))
	started, err := svc.Start(t.Context(), StartRequest{RemoteIP: "203.0.113.7"})
	require.NoError(t, err)
	otherWorkspace := workspace
	otherWorkspace.ID = uid.New(uid.WorkspacePrefix)
	other := adminPrincipal(otherWorkspace, p.Subject.ID)

	for i := range 60 {
		caller := p
		if i%2 == 1 {
			caller = other
		}
		code, err := newUserCode()
		require.NoError(t, err)
		require.NotEqual(t, started.UserCode, code)
		switch i % 3 {
		case 0:
			_, err = svc.Get(t.Context(), GetRequest{Principal: caller, UserCode: code})
		case 1:
			err = svc.Approve(t.Context(), ApproveRequest{Principal: caller, UserCode: code, Permissions: caller.Permissions})
		case 2:
			err = svc.Deny(t.Context(), DenyRequest{Principal: caller, UserCode: code})
		}
		require.Equal(t, "This login code was not found. Run unkey login again.", fault.UserFacingMessage(err))
	}

	_, err = svc.Get(t.Context(), GetRequest{Principal: p, UserCode: started.UserCode})
	require.Equal(t, codes.User.TooManyRequests.UserRateLimited.URN(), codeOf(t, err))
	err = svc.Approve(t.Context(), ApproveRequest{Principal: p, UserCode: started.UserCode, Permissions: p.Permissions})
	require.Equal(t, codes.User.TooManyRequests.UserRateLimited.URN(), codeOf(t, err))
	err = svc.Deny(t.Context(), DenyRequest{Principal: p, UserCode: started.UserCode})
	require.Equal(t, codes.User.TooManyRequests.UserRateLimited.URN(), codeOf(t, err))

	otherUser := adminPrincipal(workspace, uid.New(uid.TestPrefix))
	login, err := svc.Get(t.Context(), GetRequest{Principal: otherUser, UserCode: started.UserCode})
	require.NoError(t, err)
	require.Equal(t, "pending", login.Status)

	h.Clock.Tick(10*time.Minute + time.Second)
	login, err = svc.Get(t.Context(), GetRequest{Principal: p, UserCode: started.UserCode})
	require.NoError(t, err)
	require.Equal(t, "expired", login.Status)
}

type unavailableCodeLimiter struct{}

func (unavailableCodeLimiter) Ratelimit(context.Context, ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error) {
	return ratelimit.RatelimitResponse{}, errors.New("limiter unavailable")
}

func (unavailableCodeLimiter) RatelimitMany(context.Context, []ratelimit.RatelimitRequest) ([]ratelimit.RatelimitResponse, error) {
	return nil, errors.New("limiter unavailable")
}

func TestUserCodeActionsRejectUnavailableLimiters(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newService(h, mem{})
	p := adminPrincipal(h.Resources().UserWorkspace, uid.New(uid.TestPrefix))
	started, err := svc.Start(t.Context(), StartRequest{RemoteIP: "203.0.113.9"})
	require.NoError(t, err)

	svc.Ratelimit = unavailableCodeLimiter{}
	_, err = svc.Get(t.Context(), GetRequest{Principal: p, UserCode: started.UserCode})
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), codeOf(t, err))
	err = svc.Approve(t.Context(), ApproveRequest{Principal: p, UserCode: started.UserCode, Permissions: p.Permissions})
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), codeOf(t, err))
	err = svc.Deny(t.Context(), DenyRequest{Principal: p, UserCode: started.UserCode})
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), codeOf(t, err))

	svc.Ratelimit = h.Ratelimit
	login, err := svc.Get(t.Context(), GetRequest{Principal: p, UserCode: started.UserCode})
	require.NoError(t, err)
	require.Equal(t, "pending", login.Status)
}

func TestApproveRejectsMaliciousAndOverscopedPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	developer := developerPrincipal(workspace, uid.New(uid.TestPrefix))
	allowed := permissionOnlyDeveloperHas(t, workspace.ID)

	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.7", UserAgent: "unkey-cli/1.0"})
	require.NoError(t, err)

	err = svc.Approve(t.Context(), ApproveRequest{Principal: developer, UserCode: started.UserCode, Permissions: []string{"not a permission"}})
	require.Equal(t, codes.App.Validation.InvalidInput.URN(), codeOf(t, err))

	err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   developer,
		UserCode:    started.UserCode,
		Permissions: []string{"unkey:v1:" + uid.New(uid.WorkspacePrefix) + ":projects/*/keyspaces/*/keys/*#read"},
	})
	require.Equal(t, codes.App.Validation.InvalidInput.URN(), codeOf(t, err))

	err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   developer,
		UserCode:    started.UserCode,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#write"},
	})
	require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), codeOf(t, err))

	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal:   developer,
		UserCode:    started.UserCode,
		Permissions: []string{allowed},
	}))
}

func TestPollIssuesASessionInsteadOfARootKey(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	allowed := permissionOnlyDeveloperHas(t, workspace.ID)
	userID := uid.New(uid.TestPrefix)
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.8", UserAgent: "unkey-cli/1.0"})
	require.NoError(t, err)
	require.Equal(t, "http://localhost:3000/cli/device?user_code="+started.UserCode, started.VerificationURL)
	require.Equal(t, int32(deviceTTL/time.Second), started.ExpiresIn)
	require.EqualValues(t, pollIntervalSeconds, started.Interval)

	pending, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID, RemoteIP: "127.0.0.1", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	require.Equal(t, PollPermissionsRequired, pending.Status)
	require.Empty(t, pending.AccessToken)

	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    started.UserCode,
		Permissions: []string{allowed},
	}))

	done, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID, RemoteIP: "127.0.0.1", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	require.Equal(t, PollComplete, done.Status)
	require.NotEmpty(t, done.AccessToken)
	require.NotEmpty(t, done.RefreshToken)
	require.Equal(t, int64(accessTTL/time.Second), done.ExpiresIn)
	require.Equal(t, workspace.ID, done.WorkspaceID)

	verifier, err := tokenjwt.NewHS256Verifier[clisession.Claims]([]byte(testSecret),
		tokenjwt.WithIssuer(clisession.Issuer),
		tokenjwt.WithAudience(clisession.Audience),
		tokenjwt.WithClock(h.Clock),
	)
	require.NoError(t, err)
	claims, err := verifier.Verify(done.AccessToken)
	require.NoError(t, err)
	require.Equal(t, userID, claims.Subject)
	require.Equal(t, workspace.ID, claims.WorkspaceID)
	require.Equal(t, done.SessionID, claims.SessionID)
	require.Equal(t, []string{allowed}, claims.Permissions)
	require.Equal(t, clisession.TokenUse, claims.TokenUse)

	_, err = svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID})
	require.Error(t, err)
	require.Contains(t, fault.UserFacingMessage(err), "already issued a CLI session")
}

func TestRefreshNarrowsOnDemotionAndFailsOnRemoval(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	membership := mem{roles: map[string][]string{}, gone: map[string]bool{}}
	svc := newService(h, membership)
	allowed := permissionOnlyDeveloperHas(t, workspace.ID)
	demotedID := uid.New(uid.TestPrefix)
	tokens := login(t, svc, workspace, demotedID, allowed)

	membership.roles[demotedID] = []string{"viewer"}
	_, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), codeOf(t, err))

	row, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.True(t, row.RevokedAt.Valid)

	keptID := uid.New(uid.TestPrefix)
	membership.roles[keptID] = []string{"developer"}
	kept := login(t, svc, workspace, keptID, allowed)
	refreshed, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: kept.RefreshToken})
	require.NoError(t, err)
	require.NotEqual(t, kept.RefreshToken, refreshed.RefreshToken)

	membership.gone[keptID] = true
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: refreshed.RefreshToken})
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), codeOf(t, err))
	row, err = db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), kept.SessionID)
	require.NoError(t, err)
	require.True(t, row.RevokedAt.Valid)
}

func TestReusedRefreshTokenRevokesTheSession(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	refreshed, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)

	h.Clock.Tick(refreshReuseGrace + time.Millisecond)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))

	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: refreshed.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
}

func TestRefreshTokenTwoRotationsOldRevokesTheFamily(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	first, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)
	h.Clock.Tick(refreshReuseGrace + time.Millisecond)
	second, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: first.RefreshToken})
	require.NoError(t, err)

	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
	require.Contains(t, fault.UserFacingMessage(err), "session was revoked")

	row, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.True(t, row.RevokedAt.Valid)

	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: second.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
}

func TestUnknownTokenOfARevokedFamilyIsPlainlyInvalid(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))
	family, ok := refreshTokenFamily(tokens.RefreshToken)
	require.True(t, ok)

	_, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: newRefreshToken(newRefreshFamily())})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
	require.Equal(t, "The refresh token is invalid. Run unkey login again.", fault.UserFacingMessage(err))

	require.NoError(t, svc.Revoke(t.Context(), RevokeRequest{
		Principal: adminPrincipal(workspace, uid.New(uid.TestPrefix)),
		SessionID: tokens.SessionID,
	}))
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: newRefreshToken(family)})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
	require.Equal(t, "The refresh token is invalid. Run unkey login again.", fault.UserFacingMessage(err))
}

func TestPreviousRefreshTokenWithinGraceDoesNotRevokeTheFamily(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	refreshed, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)

	h.Clock.Tick(refreshReuseGrace)
	again, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)
	require.NotEqual(t, refreshed.RefreshToken, again.RefreshToken)

	row, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.False(t, row.RevokedAt.Valid)

	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: again.RefreshToken})
	require.NoError(t, err)
}

type overlappingMemberships struct {
	arrived sync.WaitGroup
	mu      sync.Mutex
	calls   int
}

func (m *overlappingMemberships) ActiveRoles(context.Context, string, string) ([]string, bool, error) {
	m.mu.Lock()
	m.calls++
	wait := m.calls <= 2
	m.mu.Unlock()
	if wait {
		m.arrived.Done()
		m.arrived.Wait()
	}
	return []string{"admin"}, true, nil
}

func TestOverlappingRefreshesHonorTheGraceWindow(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))
	memberships := &overlappingMemberships{}
	memberships.arrived.Add(2)
	svc.Memberships = memberships
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
			results <- err
		}()
	}
	for range 2 {
		require.NoError(t, <-results)
	}
	row, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.False(t, row.RevokedAt.Valid)
}

func TestExpiredAndRevokedSessionsCannotRefresh(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	require.NoError(t, svc.Revoke(t.Context(), RevokeRequest{
		Principal: adminPrincipal(workspace, uid.New(uid.TestPrefix)),
		SessionID: tokens.SessionID,
	}))
	_, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))

	other := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))
	h.Clock.Tick(sessionIdle + time.Second)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: other.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
}

func TestRefreshMovesTheIdleDeadline(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	row, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.Equal(t, row.CreatedAt+sessionIdle.Milliseconds(), row.ExpiresAt)

	h.Clock.Tick(24 * time.Hour)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)
	row, err = db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.Equal(t, h.Clock.Now().UnixMilli()+sessionIdle.Milliseconds(), row.ExpiresAt)
}

func TestActiveSessionEndsAtSessionMax(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))
	row, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	deadline := row.CreatedAt + sessionMax.Milliseconds()

	refreshToken := tokens.RefreshToken
	for h.Clock.Now().Add(sessionIdle-24*time.Hour).UnixMilli() < deadline {
		h.Clock.Tick(sessionIdle - 24*time.Hour)
		refreshed, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: refreshToken})
		require.NoError(t, err)
		refreshToken = refreshed.RefreshToken
		row, err = db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
		require.NoError(t, err)
		require.LessOrEqual(t, row.ExpiresAt, deadline)
	}
	require.Equal(t, deadline, row.ExpiresAt)

	h.Clock.Set(time.UnixMilli(deadline))
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: refreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
	require.Contains(t, fault.UserFacingMessage(err), "no longer valid")
}

func TestUserNameSurvivesRefresh(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	refreshed, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)

	verifier, err := tokenjwt.NewHS256Verifier[clisession.Claims]([]byte(testSecret),
		tokenjwt.WithIssuer(clisession.Issuer),
		tokenjwt.WithAudience(clisession.Audience),
		tokenjwt.WithClock(h.Clock),
	)
	require.NoError(t, err)
	claims, err := verifier.Verify(refreshed.AccessToken)
	require.NoError(t, err)
	require.Equal(t, "Admin", claims.Name)

	names := map[string]string{}
	for _, log := range h.FindAuditLogsByTargetID(t.Context(), t, tokens.SessionID) {
		names[log.Event] = log.Actor.Name
	}
	require.Equal(t, "Admin", names[string(auditlog.CLISessionRefreshEvent)])
}

func TestRootKeyCannotApprove(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop"})
	require.NoError(t, err)

	rootKey := adminPrincipal(workspace, uid.New(uid.KeyPrefix))
	rootKey.Type = principal.TypeAPIKey
	rootKey.Subject.Type = principal.SubjectTypeRootKey
	rootKey.Source = principal.KeySource{}
	_, err = svc.Get(t.Context(), GetRequest{Principal: rootKey, UserCode: started.UserCode})
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), codeOf(t, err))
	err = svc.Approve(t.Context(), ApproveRequest{Principal: rootKey, UserCode: started.UserCode, Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)}})
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), codeOf(t, err))
}

func TestDisabledServiceRejectsStart(t *testing.T) {
	svc := &Service{DashboardBaseURL: "http://localhost:3000"}
	require.False(t, svc.Enabled())
	_, err := svc.Start(context.Background(), StartRequest{DeviceName: "laptop"})
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), codeOf(t, err))
}

func login(t *testing.T, svc *Service, workspace db.Workspace, userID string, permission string) Tokens {
	t.Helper()
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.9", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    started.UserCode,
		Permissions: []string{permission},
	}))
	done, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID})
	require.NoError(t, err)
	require.Equal(t, PollComplete, done.Status)
	return Tokens{
		AccessToken:  done.AccessToken,
		RefreshToken: done.RefreshToken,
		ExpiresIn:    done.ExpiresIn,
		WorkspaceID:  done.WorkspaceID,
		SessionID:    done.SessionID,
	}
}

func codeOf(t *testing.T, err error) codes.URN {
	t.Helper()
	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok, fault.UserFacingMessage(err))
	return code
}

func TestRevokeAndListAreScopedToTheCaller(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	permission := permissionOnlyDeveloperHas(t, workspace.ID)
	ownerID := uid.New(uid.TestPrefix)
	teammateID := uid.New(uid.TestPrefix)
	owned := login(t, svc, workspace, ownerID, permission)
	teammate := login(t, svc, workspace, teammateID, permission)

	err := svc.Revoke(t.Context(), RevokeRequest{
		Principal: developerPrincipal(workspace, teammateID),
		SessionID: owned.SessionID,
	})
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), codeOf(t, err))
	row, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), owned.SessionID)
	require.NoError(t, err)
	require.False(t, row.RevokedAt.Valid)

	err = svc.Revoke(t.Context(), RevokeRequest{
		Principal: adminPrincipal(h.Resources().RootWorkspace, uid.New(uid.TestPrefix)),
		SessionID: owned.SessionID,
	})
	require.Equal(t, codes.App.Validation.InvalidInput.URN(), codeOf(t, err))
	require.Contains(t, fault.UserFacingMessage(err), "not found")
	row, err = db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), owned.SessionID)
	require.NoError(t, err)
	require.False(t, row.RevokedAt.Valid)

	sessions, err := svc.List(t.Context(), ListRequest{Principal: developerPrincipal(workspace, teammateID)})
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, teammate.SessionID, sessions[0].ID)

	sessions, err = svc.List(t.Context(), ListRequest{Principal: adminPrincipal(workspace, uid.New(uid.TestPrefix))})
	require.NoError(t, err)
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.ID)
	}
	require.Contains(t, ids, owned.SessionID)
	require.Contains(t, ids, teammate.SessionID)

	admin := adminPrincipal(workspace, uid.New(uid.TestPrefix))
	require.NoError(t, svc.Revoke(t.Context(), RevokeRequest{Principal: admin, SessionID: owned.SessionID}))
	require.NoError(t, svc.Revoke(t.Context(), RevokeRequest{Principal: admin, SessionID: owned.SessionID}))
	row, err = db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), owned.SessionID)
	require.NoError(t, err)
	require.True(t, row.RevokedAt.Valid)
	logs := h.FindAuditLogsByTargetID(t.Context(), t, owned.SessionID)
	found := false
	for _, log := range logs {
		if log.Event != "cli.session.revoke" {
			continue
		}
		found = true
		require.Equal(t, admin.Subject.ID, log.Actor.ID)
		require.NotContains(t, log.Actor.Meta, "session_id")
		require.NotContains(t, log.Actor.Meta, "device")
	}
	require.True(t, found)
}

func TestConcurrentPollsIssueOneSession(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	userID := uid.New(uid.TestPrefix)
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.10", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    started.UserCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	}))

	const pollers = 8
	results := make([]PollResult, pollers)
	var wg sync.WaitGroup
	for i := range pollers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, pollErr := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID})
			if pollErr == nil {
				results[i] = result
			}
		}()
	}
	wg.Wait()

	completed := 0
	for _, result := range results {
		if result.Status == PollComplete {
			completed++
		}
	}
	require.Equal(t, 1, completed)

	rows, err := db.Query.ListCLISessionsByUser(t.Context(), h.DB.RW(), db.ListCLISessionsByUserParams{
		WorkspaceID: workspace.ID,
		UserID:      userID,
		Now:         h.Clock.Now().UnixMilli(),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestSecondApproverCannotTakeOverAnApprovedLogin(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	permission := permissionOnlyDeveloperHas(t, workspace.ID)
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.11", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, uid.New(uid.TestPrefix)),
		UserCode:    started.UserCode,
		Permissions: []string{permission},
	}))

	err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, uid.New(uid.TestPrefix)),
		UserCode:    started.UserCode,
		Permissions: []string{permission},
	})
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), codeOf(t, err))
}

func TestDeniedLoginStaysDenied(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	developer := developerPrincipal(workspace, uid.New(uid.TestPrefix))
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.12", UserAgent: "unkey-cli"})
	require.NoError(t, err)

	require.NoError(t, svc.Deny(t.Context(), DenyRequest{Principal: developer, UserCode: started.UserCode}))
	polled, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID})
	require.NoError(t, err)
	require.Equal(t, PollDenied, polled.Status)
	require.Empty(t, polled.AccessToken)

	err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   developer,
		UserCode:    started.UserCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	})
	require.Equal(t, codes.App.Validation.InvalidInput.URN(), codeOf(t, err))
}

func TestPollAfterDeviceTTLReportsExpired(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newService(h, mem{})
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.13", UserAgent: "unkey-cli"})
	require.NoError(t, err)

	h.Clock.Tick(deviceTTL + time.Second)
	polled, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID})
	require.NoError(t, err)
	require.Equal(t, PollExpired, polled.Status)
	require.Empty(t, polled.AccessToken)
}

func TestApproverDemotedBeforePollIsDenied(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	membership := mem{roles: map[string][]string{}, gone: map[string]bool{}}
	svc := newService(h, membership)
	userID := uid.New(uid.TestPrefix)
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.14", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    started.UserCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	}))

	membership.roles[userID] = []string{"viewer"}
	polled, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID})
	require.NoError(t, err)
	require.Equal(t, PollDenied, polled.Status)
	require.Empty(t, polled.AccessToken)

	rows, err := db.Query.ListCLISessionsByUser(t.Context(), h.DB.RW(), db.ListCLISessionsByUserParams{
		WorkspaceID: workspace.ID,
		UserID:      userID,
		Now:         h.Clock.Now().UnixMilli(),
	})
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestApprovalChangedDetectsReapproval(t *testing.T) {
	workspaceID := sql.NullString{String: uid.New(uid.WorkspacePrefix), Valid: true}
	approverID := sql.NullString{String: uid.New(uid.TestPrefix), Valid: true}
	granted, err := json.Marshal([]string{"granted"})
	require.NoError(t, err)
	regranted, err := json.Marshal([]string{"regranted"})
	require.NoError(t, err)
	read := db.DeviceAuthorization{
		WorkspaceID:    workspaceID,
		ApproverUserID: approverID,
		Permissions:    granted,
		ApprovedAt:     sql.NullInt64{Int64: 1, Valid: true},
	}
	require.False(t, approvalChanged(read, read))

	tests := []struct {
		name   string
		mutate func(db.DeviceAuthorization) db.DeviceAuthorization
	}{
		{name: "approved again", mutate: func(r db.DeviceAuthorization) db.DeviceAuthorization {
			r.ApprovedAt = sql.NullInt64{Int64: 2, Valid: true}
			return r
		}},
		{name: "different workspace", mutate: func(r db.DeviceAuthorization) db.DeviceAuthorization {
			r.WorkspaceID = sql.NullString{String: uid.New(uid.WorkspacePrefix), Valid: true}
			return r
		}},
		{name: "different approver", mutate: func(r db.DeviceAuthorization) db.DeviceAuthorization {
			r.ApproverUserID = sql.NullString{String: uid.New(uid.TestPrefix), Valid: true}
			return r
		}},
		{name: "different permissions", mutate: func(r db.DeviceAuthorization) db.DeviceAuthorization {
			r.Permissions = regranted
			return r
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.True(t, approvalChanged(read, tt.mutate(read)))
		})
	}
}
