package auth

import (
	"context"
	"log/slog"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Authenticator is what API handlers depend on. They never see the resolver
// chain or any scheme-specific detail; they ask Authenticate and get back a
// Principal (plus a deferred emit). Tests substitute their own fake.
//
// Authenticate guarantees the returned Emit is never nil, on every path,
// so handlers can `defer emit()` immediately after the call without a
// nil check. Implementations must preserve this guarantee.
type Authenticator interface {
	Authenticate(ctx context.Context, sess *zen.Session) (*Principal, Emit, error)
}

// Emit is the deferred audit/telemetry callback returned alongside a
// Principal. Handlers run it via defer so the audit event records the
// final outcome (status code, duration). Schemes without per-request
// telemetry return EmptyEmit.
type Emit = func()

// EmptyEmit is the no-op Emit used by schemes that have no audit/telemetry
// to flush.
var EmptyEmit Emit = func() {}

// New builds an Authenticator backed by the given Resolver chain. The
// resolvers are tried in order; the first to return a non-nil Principal
// wins, the first to return an error stops the walk. Callers should put
// the cheaper / more specific schemes first.
func New(resolvers ...Resolver) Authenticator {
	return chain(resolvers)
}

// chain is the canonical Authenticator implementation: a slice of resolvers
// walked in order. It is unexported so callers depend on the Authenticator
// interface, not on the chain type.
type chain []Resolver

func (c chain) Authenticate(ctx context.Context, sess *zen.Session) (*Principal, Emit, error) {
	for _, r := range c {
		p, emit, err := r.Resolve(ctx, sess)
		if err != nil {
			return nil, EmptyEmit, err
		}
		if p == nil {
			continue
		}
		sess.WorkspaceID = p.WorkspaceID
		logger.Set(ctx, slog.Group("auth",
			slog.String("scheme", string(p.Scheme)),
			slog.String("workspace_id", p.WorkspaceID),
			slog.String("subject_id", p.ID),
			slog.String("subject_name", p.DisplayName),
			slog.Int("permission_count", len(p.Permissions)),
		))
		// Normalize so handlers can `defer emit()` unconditionally without
		// a nil check, regardless of what the resolver returned.
		if emit == nil {
			emit = EmptyEmit
		}
		return p, emit, nil
	}
	// No resolver claimed the request. If zen.Bearer can't parse the
	// Authorization header at all (missing entirely, missing "Bearer "
	// prefix, empty value), surface that error directly so the client
	// gets a precise diagnosis (Missing → 400, Malformed → 400). The
	// root-key resolver is the catch-all otherwise, so reaching this
	// fallback means the header was malformed at the parser level.
	if _, bearerErr := zen.Bearer(sess); bearerErr != nil {
		return nil, EmptyEmit, bearerErr
	}
	return nil, EmptyEmit, fault.New("no credentials",
		fault.Code(codes.Auth.Authentication.Missing.URN()),
		fault.Internal("no resolver matched the request"),
		fault.Public("You must provide a bearer credential in the Authorization header."),
	)
}
