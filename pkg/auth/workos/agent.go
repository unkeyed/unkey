package workos

import (
	"context"
	"strings"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

// agentRegistrationPrefix marks WorkOS agent registration subjects. The AuthKit
// issuer also mints tokens for other subjects, which are not agents.
const agentRegistrationPrefix = "agent_reg_"

// Memberships resolves a user's current roles in a WorkOS organization.
// found is false when the user has no active membership. A non-nil error is an
// infrastructure failure, not a missing membership.
type Memberships interface {
	ActiveRoles(ctx context.Context, orgID string, userID string) (roles []string, found bool, err error)
}

// MembershipKey identifies one user's membership in one WorkOS organization.
type MembershipKey struct {
	OrgID  string
	UserID string
}

// Membership is the cached answer for a [MembershipKey]. Found is false when
// the user has no active membership in the organization.
type Membership struct {
	Roles []string
	Found bool
}

// NewAgentResolver turns verified WorkOS agent access tokens into principals
// that act for the user who claimed the agent registration.
//
// inner must verify the token and map its organization to a workspace. The
// returned resolver grants the claiming user's live roles in that organization,
// read through membershipCache, so demoting or removing the user reaches the
// agent once the cached membership expires. Unclaimed registrations and users
// without an active membership are refused. The token's scope claim is not
// read; agents get the full access of their user's roles.
func NewAgentResolver(inner auth.Resolver, memberships Memberships, membershipCache cache.Cache[MembershipKey, Membership]) (auth.Resolver, error) {
	if err := assert.All(
		assert.NotNil(inner, "inner resolver is required"),
		assert.NotNil(memberships, "WorkOS memberships are required"),
		assert.NotNil(membershipCache, "membership cache is required"),
	); err != nil {
		return nil, err
	}

	return agentResolver{
		resolver:        inner,
		memberships:     memberships,
		membershipCache: membershipCache,
	}, nil
}

// agentResolver maps verified WorkOS agent tokens to principals for the
// claiming user.
type agentResolver struct {
	resolver        auth.Resolver
	memberships     Memberships
	membershipCache cache.Cache[MembershipKey, Membership]
}

// Resolve delegates token verification and then replaces JWT principals with
// agent principals for the claiming user.
func (r agentResolver) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	p, err := r.resolver.Resolve(ctx, sess)
	if err != nil || p == nil {
		return p, err
	}
	if p.Type != principal.TypeJWT {
		return p, nil
	}

	source, ok := p.Source.(principal.JWTSource)
	if err := assert.True(ok, "JWT principal must have a JWT source"); err != nil {
		return nil, err
	}

	registrationID, _ := source.Payload["sub"].(string)
	if !strings.HasPrefix(registrationID, agentRegistrationPrefix) {
		return nil, fault.New("not an agent token",
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("JWT subject is not a WorkOS agent registration"),
			fault.Public("Invalid bearer token."),
		)
	}

	userID := actorSubject(source.Payload)
	if userID == "" {
		return nil, fault.New("unclaimed agent registration",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("agent token has no act.sub claim"),
			fault.Public("This agent registration has not been claimed by a user. A user must claim it through the service_auth claim ceremony before the agent can call the Unkey API."),
		)
	}

	key := MembershipKey{OrgID: source.OrgID, UserID: userID}
	membership, _, err := r.membershipCache.SWR(ctx, key, func(ctx context.Context) (Membership, error) {
		roles, found, lookupErr := r.memberships.ActiveRoles(ctx, key.OrgID, key.UserID)
		return Membership{Roles: roles, Found: found}, lookupErr
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load the agent user's WorkOS membership"),
			fault.Public("Unable to verify the bearer token right now."),
		)
	}
	if !membership.Found {
		return nil, fault.New("agent user is not a workspace member",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("the user who claimed the agent has no active membership in the token organization"),
			fault.Public("The user who claimed this agent is not a member of this workspace."),
		)
	}

	return &principal.Principal{
		Version: p.Version,
		Subject: principal.Subject{
			ID:   userID,
			Name: userID,
			Type: principal.SubjectTypeUser,
		},
		Type: principal.TypeJWT,
		Source: principal.AgentSource{
			RegistrationID: registrationID,
			ExpiresAt:      source.ExpiresAt,
		},
		AuthorizedWorkspaceID: p.AuthorizedWorkspaceID,
		Permissions:           PermissionsForRoles(p.AuthorizedWorkspaceID, membership.Roles),
	}, nil
}

// actorSubject returns the act.sub claim, the user who claimed the agent
// registration, or "" when the token has no actor.
func actorSubject(payload map[string]any) string {
	actor, _ := payload["act"].(map[string]any)
	userID, _ := actor["sub"].(string)
	return userID
}
