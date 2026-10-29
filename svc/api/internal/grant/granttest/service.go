// Package granttest builds a [grant.Service] for route tests.
package granttest

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/cache"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

const signingSecret = "test-cli-signing-secret-32-bytes!"

type allowAdmin struct{}

func (allowAdmin) ActiveRoles(context.Context, string, string) ([]string, bool, error) {
	return []string{"admin"}, true, nil
}

// New returns a service that treats every user as an admin member.
func New(h *testutil.Harness) *grant.Service {
	signer, err := tokenjwt.NewHS256Signer[clisession.Claims]([]byte(signingSecret))
	if err != nil {
		panic(err)
	}
	return &grant.Service{
		DB:               h.DB,
		Auditlogs:        h.Auditlogs,
		Clock:            h.Clock,
		Ratelimit:        h.Ratelimit,
		Memberships:      allowAdmin{},
		DashboardBaseURL: "http://dashboard.test",
		Signer:           signer,
		SessionCache:     cache.NewNoopCache[string, grant.SessionVerdict](),
	}
}
