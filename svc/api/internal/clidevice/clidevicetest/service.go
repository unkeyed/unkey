// Package clidevicetest builds a [clidevice.Service] for route tests.
package clidevicetest

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth/clisession"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

const signingSecret = "test-cli-signing-secret-32-bytes!"

type allowAdmin struct{}

func (allowAdmin) ActiveRoles(context.Context, string, string) ([]string, bool, error) {
	return []string{"admin"}, true, nil
}

// New returns a service that treats every user as an admin member.
func New(h *testutil.Harness) *clidevice.Service {
	signer, err := tokenjwt.NewHS256Signer[clisession.Claims]([]byte(signingSecret))
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
		SessionCache:     nil,
	}
}
