package clidevice

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
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
		Memberships:      membership,
		DashboardBaseURL: "http://localhost:3000",
		Signer:           signer,
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

func TestApproveRejectsMissingMaliciousAndOverscopedPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	developer := developerPrincipal(workspace, "user_dev")
	allowed := permissionOnlyDeveloperHas(t, workspace.ID)

	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.7", UserAgent: "unkey-cli/1.0"})
	require.NoError(t, err)

	err = svc.Approve(t.Context(), ApproveRequest{Principal: developer, UserCode: started.UserCode, Permissions: nil})
	require.Equal(t, codes.App.Validation.InvalidInput.URN(), codeOf(t, err))

	err = svc.Approve(t.Context(), ApproveRequest{Principal: developer, UserCode: started.UserCode, Permissions: []string{"not a permission"}})
	require.Equal(t, codes.App.Validation.InvalidInput.URN(), codeOf(t, err))

	err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   developer,
		UserCode:    started.UserCode,
		Permissions: []string{"unkey:v1:ws_other:projects/*/keyspaces/*/keys/*#read"},
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
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.8", UserAgent: "unkey-cli/1.0"})
	require.NoError(t, err)

	pending, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID, RemoteIP: "127.0.0.1", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	require.Equal(t, PollPermissionsRequired, pending.Status)
	require.Empty(t, pending.AccessToken)

	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, "user_dev"),
		UserCode:    started.UserCode,
		Permissions: []string{allowed},
	}))

	done, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID, RemoteIP: "127.0.0.1", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	require.Equal(t, PollComplete, done.Status)
	require.NotEmpty(t, done.AccessToken)
	require.NotEmpty(t, done.RefreshToken)
	require.Equal(t, int64(time.Hour/time.Second), done.ExpiresIn)
	require.Equal(t, workspace.ID, done.WorkspaceID)

	verifier, err := tokenjwt.NewHS256Verifier[clisession.Claims]([]byte(testSecret),
		tokenjwt.WithIssuer(clisession.Issuer),
		tokenjwt.WithAudience(clisession.Audience),
		tokenjwt.WithClock(h.Clock),
	)
	require.NoError(t, err)
	claims, err := verifier.Verify(done.AccessToken)
	require.NoError(t, err)
	require.Equal(t, "user_dev", claims.Subject)
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
	tokens := login(t, svc, workspace, "user_dev", allowed)

	membership.roles["user_dev"] = []string{"viewer"}
	_, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), codeOf(t, err))

	row, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.True(t, row.RevokedAt.Valid)

	membership.roles["user_kept"] = []string{"developer"}
	kept := login(t, svc, workspace, "user_kept", allowed)
	refreshed, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: kept.RefreshToken})
	require.NoError(t, err)
	require.NotEqual(t, kept.RefreshToken, refreshed.RefreshToken)

	membership.gone["user_kept"] = true
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
	tokens := login(t, svc, workspace, "user_dev", permissionOnlyDeveloperHas(t, workspace.ID))

	refreshed, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)

	h.Clock.Tick(refreshReuseGrace + time.Millisecond)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))

	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: refreshed.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
}

func TestPreviousRefreshTokenWithinGraceDoesNotRevokeTheFamily(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, "user_dev", permissionOnlyDeveloperHas(t, workspace.ID))

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

func TestExpiredAndRevokedSessionsCannotRefresh(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	tokens := login(t, svc, workspace, "user_dev", permissionOnlyDeveloperHas(t, workspace.ID))

	require.NoError(t, svc.Revoke(t.Context(), RevokeRequest{
		Principal: adminPrincipal(workspace, "user_admin"),
		SessionID: tokens.SessionID,
	}))
	_, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))

	other := login(t, svc, workspace, "user_other", permissionOnlyDeveloperHas(t, workspace.ID))
	h.Clock.Tick(refreshIdle + time.Second)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: other.RefreshToken})
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
}

func TestRootKeyCannotApprove(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newService(h, mem{})
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop"})
	require.NoError(t, err)

	rootKey := adminPrincipal(workspace, "key_1")
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
