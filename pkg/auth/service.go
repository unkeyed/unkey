package auth

import (
	"context"
	"log/slog"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Authenticator authenticates requests into normalized principals.
type Authenticator interface {
	// Authenticate resolves the request credential into a normalized principal.
	// Implementations try configured credential sources in deterministic order
	// and return the first principal whose resolver claims the request.
	Authenticate(ctx context.Context, sess *zen.Session) (*principal.Principal, error)
}

// New builds an authenticator that tries each resolver in the order provided.
func New(resolvers ...Resolver) Authenticator {
	return chain(resolvers)
}

// chain is the ordered resolver list behind [New]; earlier resolvers win.
type chain []Resolver

// Authenticate resolves credentials by asking each resolver to claim the request.
func (c chain) Authenticate(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	// Reuse a principal that was already resolved earlier in the request.
	if p, err := sess.GetPrincipal(); err == nil {
		return p, nil
	}

	// A resolver error does not abort the chain. Multiple resolvers can accept
	// the same credential shape (for example HS256-secret and JWKS jwt entries
	// both claim JWT-shaped bearers), so a rejection by one entry must not block
	// an entry that can verify the token. Every resolver fully verifies before
	// claiming a request, so a later resolver accepting a credential an earlier
	// one rejected is a valid authentication under its own trust configuration,
	// not a bypass.
	//
	// When nothing resolves, which error is reported matters, and it must not
	// depend on resolver order. The chain keeps the most authoritative rejection
	// by severity (see [errorSeverity]): an infrastructure failure (a resolver
	// that could not determine validity, so the client should retry) outranks an
	// authorization failure (a resolver that verified the credential but denied
	// it), which outranks an authentication rejection (a resolver that could not
	// verify the credential, leaving open that another resolver could). Without
	// this, a sibling resolver's 401 could mask a 403 from the resolver that
	// actually verified the token, or a 503 outage, turning either into a forced
	// logout. Ties keep the first error for deterministic reporting.
	var bestErr error
	bestSeverity := -1
	for _, resolver := range c {
		principal, err := resolver.Resolve(ctx, sess)
		if err != nil {
			if severity := errorSeverity(err); severity > bestSeverity {
				bestErr, bestSeverity = err, severity
			}
			continue
		}
		if principal == nil {
			continue
		}

		sess.SetPrincipal(principal)
		logger.Set(ctx, slog.Group("auth",
			slog.String("type", string(principal.Type)),
			slog.String("workspace_id", principal.WorkspaceID),
			slog.String("subject", principal.Subject.ID),
			slog.String("subject_name", principal.Subject.Name),
			slog.Int("permission_count", len(principal.Permissions)),
		))

		return principal, nil
	}

	if bestErr != nil {
		return nil, bestErr
	}

	// Preserve Bearer's specific malformed-header errors after every resolver
	// declined the request. A syntactically valid bearer that no resolver claims
	// falls through to the generic missing-credentials response below.
	if _, err := zen.Bearer(sess); err != nil {
		return nil, err
	}

	return nil, fault.New("no credentials",
		fault.Code(codes.Auth.Authentication.Missing.URN()),
		fault.Internal("no resolver matched the request"),
		fault.Public("You must provide a bearer credential in the Authorization header."),
	)
}

// Resolver rejection severities, ordered so a more authoritative answer about
// the credential outranks a less authoritative one when the chain reports why
// nothing resolved. See [errorSeverity].
const (
	// severityOther covers uncoded errors and any code outside the buckets
	// below. It is the floor so a classified rejection always wins.
	severityOther = iota
	// severityAuthentication is a credential rejection (401): this resolver
	// could not verify the credential, but another might, so it is the weakest
	// real signal.
	severityAuthentication
	// severityAuthorization is a post-verification denial (403): the resolver
	// verified the credential and then refused it, so it is authoritative about
	// the credential in a way an authentication rejection is not.
	severityAuthorization
	// severityInfrastructure is an availability failure (5xx): the resolver
	// could not determine validity at all, so the client should retry rather
	// than treat the credential as bad. It outranks every refusal.
	severityInfrastructure
)

// errorSeverity ranks a resolver rejection so [chain.Authenticate] can report
// the most authoritative one independent of resolver order. Resolvers tag
// their failures with fault codes: availability failures (JWKS endpoint outage,
// workspace lookup failure) as the internal application codes that map to 5xx,
// authorization denials under the authorization category, and credential
// rejections under the authentication category.
func errorSeverity(err error) int {
	urn, ok := fault.GetCode(err)
	if !ok {
		return severityOther
	}
	if urn == codes.App.Internal.ServiceUnavailable.URN() || urn == codes.App.Internal.UnexpectedError.URN() {
		return severityInfrastructure
	}
	code, parseErr := codes.ParseURN(urn)
	if parseErr != nil {
		return severityOther
	}
	if code.Category == codes.CategoryUnkeyAuthorization {
		return severityAuthorization
	}
	if code.Category == codes.CategoryUnkeyAuthentication {
		return severityAuthentication
	}
	return severityOther
}
