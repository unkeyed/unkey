package workos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

type stubMemberships struct {
	roles []string
	found bool
	err   error
	calls []MembershipKey
}

// ActiveRoles records the lookup and returns the configured membership.
func (s *stubMemberships) ActiveRoles(_ context.Context, orgID string, userID string) ([]string, bool, error) {
	s.calls = append(s.calls, MembershipKey{OrgID: orgID, UserID: userID})
	return s.roles, s.found, s.err
}

type agentFixture struct {
	workspaceID    string
	orgID          string
	userID         string
	registrationID string
	expiresAt      time.Time
}

func newAgentFixture() agentFixture {
	return agentFixture{
		workspaceID:    uid.New(uid.WorkspacePrefix),
		orgID:          uid.New(uid.TestPrefix),
		userID:         uid.New(uid.TestPrefix),
		registrationID: agentRegistrationPrefix + uid.New(uid.TestPrefix),
		expiresAt:      time.Unix(time.Now().Add(5*time.Minute).Unix(), 0),
	}
}

// claimedPayload is the payload of an agent token after a user claimed it.
func (f agentFixture) claimedPayload() map[string]any {
	return map[string]any{
		"sub":    f.registrationID,
		"org_id": f.orgID,
		"act":    map[string]any{"sub": f.userID},
	}
}

// verified returns the principal the JWT resolver produces for payload.
func (f agentFixture) verified(payload map[string]any) *principal.Principal {
	return &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: f.registrationID, Name: f.registrationID, Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Payload: payload, OrgID: f.orgID, ExpiresAt: f.expiresAt},
		AuthorizedWorkspaceID: f.workspaceID,
	}
}

func (f agentFixture) permission(resource string, action string) string {
	return "unkey:v1:" + f.workspaceID + ":" + resource + "#" + action
}

func newAgentTestResolver(t *testing.T, inner auth.Resolver, memberships Memberships) auth.Resolver {
	t.Helper()

	membershipCache, err := cache.New(cache.Config[MembershipKey, Membership]{
		Fresh:    time.Minute,
		Stale:    time.Minute,
		MaxSize:  100,
		Resource: "test_workos_agent_roles",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(membershipCache.Close)

	resolver, err := NewAgentResolver(inner, memberships, membershipCache)
	require.NoError(t, err)
	return resolver
}

func requireCode(t *testing.T, err error, want codes.URN) {
	t.Helper()

	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, want, code)
}

// TestAgentResolverActsAsClaimingUser guarantees a claimed agent token becomes
// a principal for the claiming user with that user's live role, and that a
// cached membership serves later requests without another WorkOS lookup.
func TestAgentResolverActsAsClaimingUser(t *testing.T) {
	t.Parallel()

	f := newAgentFixture()
	memberships := &stubMemberships{roles: []string{"admin"}, found: true}
	resolver := newAgentTestResolver(t, stubResolver{principal: f.verified(f.claimedPayload())}, memberships)

	p, err := resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, principal.Subject{ID: f.userID, Name: f.userID, Type: principal.SubjectTypeUser}, p.Subject)
	require.Equal(t, principal.TypeJWT, p.Type)
	require.Equal(t, principal.AgentSource{RegistrationID: f.registrationID, ExpiresAt: f.expiresAt}, p.Source)
	require.Equal(t, f.workspaceID, p.AuthorizedWorkspaceID)
	require.Equal(t, []string{f.permission("**", "*")}, p.Permissions)
	require.Equal(t, []MembershipKey{{OrgID: f.orgID, UserID: f.userID}}, memberships.calls)

	_, err = resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, memberships.calls, 1)
}

// TestAgentResolverLimitsAgentToUserRole guarantees an agent receives only its
// user's current role, even when the token carries a broader roles claim.
func TestAgentResolverLimitsAgentToUserRole(t *testing.T) {
	t.Parallel()

	f := newAgentFixture()
	verified := f.verified(f.claimedPayload())
	verified.Source = principal.JWTSource{Payload: f.claimedPayload(), OrgID: f.orgID, Roles: []string{"admin"}}
	resolver := newAgentTestResolver(t, stubResolver{principal: verified}, &stubMemberships{roles: []string{"viewer"}, found: true})

	p, err := resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Contains(t, p.Permissions, f.permission("projects/*", "read"))
	require.NotContains(t, p.Permissions, f.permission("projects/*", "write"))
	require.NotContains(t, p.Permissions, f.permission("**", "*"))
}

// TestAgentResolverPassesThroughOtherResults guarantees the agent provider
// leaves non-JWT principals, declined requests, and verification errors as the
// inner resolver returned them.
func TestAgentResolverPassesThroughOtherResults(t *testing.T) {
	t.Parallel()

	rootKey := &principal.Principal{
		Type:                  principal.TypeAPIKey,
		AuthorizedWorkspaceID: uid.New(uid.WorkspacePrefix),
		Permissions:           []string{"unkey:v1:" + uid.New(uid.WorkspacePrefix) + ":projects/*#read"},
	}
	verifyErr := errors.New("boom")

	for _, tc := range []struct {
		name  string
		inner stubResolver
	}{
		{name: "non-JWT principal", inner: stubResolver{principal: rootKey}},
		{name: "declined", inner: stubResolver{}},
		{name: "verification error", inner: stubResolver{err: verifyErr}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			memberships := &stubMemberships{}
			resolver := newAgentTestResolver(t, tc.inner, memberships)

			p, err := resolver.Resolve(context.Background(), nil)
			require.Same(t, tc.inner.principal, p)
			require.ErrorIs(t, err, tc.inner.err)
			require.Empty(t, memberships.calls)
		})
	}
}

// TestAgentResolverRejectsNonAgentSubject guarantees other tokens from the
// AuthKit issuer cannot authenticate through the agent provider.
func TestAgentResolverRejectsNonAgentSubject(t *testing.T) {
	t.Parallel()

	f := newAgentFixture()
	for _, tc := range []struct {
		name    string
		subject any
	}{
		{name: "user subject", subject: uid.New(uid.TestPrefix)},
		{name: "missing subject", subject: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload := f.claimedPayload()
			payload["sub"] = tc.subject
			memberships := &stubMemberships{roles: []string{"admin"}, found: true}
			resolver := newAgentTestResolver(t, stubResolver{principal: f.verified(payload)}, memberships)

			p, err := resolver.Resolve(context.Background(), nil)
			require.Nil(t, p)
			requireCode(t, err, codes.Auth.Authentication.Malformed.URN())
			require.Equal(t, "Invalid bearer token.", fault.UserFacingMessage(err))
			require.Empty(t, memberships.calls)
		})
	}
}

// TestAgentResolverRejectsUnclaimedRegistration guarantees an agent that no
// user has claimed is refused and told to complete the claim ceremony.
func TestAgentResolverRejectsUnclaimedRegistration(t *testing.T) {
	t.Parallel()

	f := newAgentFixture()
	for _, tc := range []struct {
		name  string
		actor any
	}{
		{name: "no actor", actor: nil},
		{name: "empty actor subject", actor: map[string]any{"sub": ""}},
		{name: "actor is not an object", actor: f.userID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload := f.claimedPayload()
			payload["act"] = tc.actor
			memberships := &stubMemberships{roles: []string{"admin"}, found: true}
			resolver := newAgentTestResolver(t, stubResolver{principal: f.verified(payload)}, memberships)

			p, err := resolver.Resolve(context.Background(), nil)
			require.Nil(t, p)
			requireCode(t, err, codes.Auth.Authorization.Forbidden.URN())
			require.Contains(t, fault.UserFacingMessage(err), "claim")
			require.Empty(t, memberships.calls)
		})
	}
}

// TestAgentResolverRejectsUserWithoutMembership guarantees an agent loses
// access when its claiming user is not an active member of the organization.
func TestAgentResolverRejectsUserWithoutMembership(t *testing.T) {
	t.Parallel()

	f := newAgentFixture()
	resolver := newAgentTestResolver(t, stubResolver{principal: f.verified(f.claimedPayload())}, &stubMemberships{found: false})

	p, err := resolver.Resolve(context.Background(), nil)
	require.Nil(t, p)
	requireCode(t, err, codes.Auth.Authorization.Forbidden.URN())
	require.Contains(t, fault.UserFacingMessage(err), "not a member")
}

// TestAgentResolverReportsMembershipLookupFailure guarantees a WorkOS outage
// is reported as retryable rather than as a bad credential, and is not cached.
func TestAgentResolverReportsMembershipLookupFailure(t *testing.T) {
	t.Parallel()

	f := newAgentFixture()
	memberships := &stubMemberships{err: errors.New("workos unavailable")}
	resolver := newAgentTestResolver(t, stubResolver{principal: f.verified(f.claimedPayload())}, memberships)

	p, err := resolver.Resolve(context.Background(), nil)
	require.Nil(t, p)
	requireCode(t, err, codes.App.Internal.ServiceUnavailable.URN())

	_, err = resolver.Resolve(context.Background(), nil)
	require.Error(t, err)
	require.Len(t, memberships.calls, 2)
}

// TestAgentResolverRejectsJWTPrincipalWithoutJWTSource guarantees a JWT
// principal without token claims fails instead of being treated as an agent.
func TestAgentResolverRejectsJWTPrincipalWithoutJWTSource(t *testing.T) {
	t.Parallel()

	f := newAgentFixture()
	verified := f.verified(f.claimedPayload())
	verified.Source = principal.CLISessionSource{}
	memberships := &stubMemberships{roles: []string{"admin"}, found: true}
	resolver := newAgentTestResolver(t, stubResolver{principal: verified}, memberships)

	p, err := resolver.Resolve(context.Background(), nil)
	require.Nil(t, p)
	requireCode(t, err, codes.App.Validation.AssertionFailed.URN())
	require.Empty(t, memberships.calls)
}
