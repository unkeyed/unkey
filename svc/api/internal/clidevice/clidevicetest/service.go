// Package clidevicetest builds a [clidevice.Service] for route tests.
package clidevicetest

import (
	"context"
	"errors"
	"slices"

	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/devicesession"
	"github.com/unkeyed/unkey/pkg/cache"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

const signingSecret = "test-cli-signing-secret-32-bytes!"

const (
	// Issuer is the OAuth issuer [NewOAuth] configures.
	Issuer = "https://api.unkey.test"
	// Resource is an OAuth resource [NewOAuth] allows besides [Issuer].
	Resource = "https://mcp.unkey.test/mcp/v2"
)

type allowAdmin struct{}

func (allowAdmin) ActiveRoles(context.Context, string, string) ([]string, bool, error) {
	return []string{"admin"}, true, nil
}

// New returns a service that treats every user as an admin member.
func New(h *testutil.Harness) *clidevice.Service {
	signer, err := tokenjwt.NewHS256Signer[devicesession.Claims]([]byte(signingSecret))
	if err != nil {
		panic(err)
	}
	return &clidevice.Service{
		DB:               h.DB,
		Auditlogs:        h.Auditlogs,
		Clock:            h.Clock,
		Ratelimit:        h.Ratelimit,
		Memberships:      allowAdmin{},
		DashboardBaseURL: "http://dashboard.test",
		Signer:           signer,
		SessionCache:     cache.NewNoopCache[string, clidevice.SessionVerdict](),
		IssuerURL:        "",
		OAuthResources:   nil,
		ClientMetadata:   nil,
	}
}

// ClientDocuments serves client metadata documents keyed by client_id.
type ClientDocuments map[string]clidevice.ClientMetadata

// Fetch returns the document registered for clientID.
func (d ClientDocuments) Fetch(_ context.Context, clientID string) (clidevice.ClientMetadata, error) {
	document, ok := d[clientID]
	if !ok {
		return clidevice.ClientMetadata{}, errors.New("client metadata not found")
	}
	return document, nil
}

// NewOAuth returns [New] with the OAuth authorization server enabled for the
// clients in documents.
func NewOAuth(h *testutil.Harness, documents ClientDocuments) *clidevice.Service {
	svc := New(h)
	svc.IssuerURL = Issuer
	svc.OAuthResources = []string{Issuer, Resource}
	svc.ClientMetadata = documents
	return svc
}

// Middleware is the harness's public stack followed by authentication that
// accepts only CLI access tokens issued by svc.
func Middleware(h *testutil.Harness, svc *clidevice.Service) []zen.Middleware {
	resolver, err := devicesession.NewResolver(svc, h.Clock, []byte(signingSecret))
	if err != nil {
		panic(err)
	}
	exemptions, err := openapi.WorkspaceRateLimitExemptionsFromSpec(openapi.Spec)
	if err != nil {
		panic(err)
	}
	return append(slices.Clone(h.PublicMiddleware()), middleware.WithAuthentication(middleware.AuthenticationConfig{
		Auth:                auth.New(resolver),
		KeyVerifications:    nil,
		Region:              "test",
		Database:            h.DB,
		LimitsCache:         h.Caches.WorkspaceLimits,
		Ratelimit:           h.Ratelimit,
		RateLimitExemptions: exemptions,
	}))
}
