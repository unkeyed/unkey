package api

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	v1Liveness "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_liveness"
	v1RatelimitCommitLease "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_commitLease"
	v1RatelimitMultiRatelimit "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_multiRatelimit"
	v1RatelimitRatelimit "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_ratelimit"
	v1VaultDecrypt "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_vault_decrypt"
	v1VaultEncrypt "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_vault_encrypt"
	v1VaultEncryptBulk "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_vault_encrypt_bulk"
)

func (s *Server) RegisterRoutes() {
	svc := routes.Services{
		Logger:           s.logger,
		Metrics:          s.metrics,
		Vault:            s.Vault,
		Ratelimit:        s.Ratelimit,
		OpenApiValidator: s.validator,
	}

	v1Liveness.New(svc).Register(s.app)
	v1RatelimitCommitLease.New(svc).
		WithMiddleware(s.BearerAuthFromSecret(s.authToken)).
		Register(s.app)
	v1RatelimitMultiRatelimit.New(svc).
		WithMiddleware(s.BearerAuthFromSecret(s.authToken)).
		Register(s.app)
	v1RatelimitRatelimit.New(svc).
		WithMiddleware(s.BearerAuthFromSecret(s.authToken)).
		Register(s.app)
	v1VaultDecrypt.New(svc).
		WithMiddleware(s.BearerAuthFromSecret(s.authToken)).
		Register(s.app)
	v1VaultEncrypt.New(svc).
		WithMiddleware(s.BearerAuthFromSecret(s.authToken)).
		Register(s.app)
	v1VaultEncryptBulk.New(svc).
		WithMiddleware(s.BearerAuthFromSecret(s.authToken)).
		Register(s.app)

}
