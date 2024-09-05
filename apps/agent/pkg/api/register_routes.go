package api

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	notFound "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/not_found"
	openapi "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/openapi"
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
		Vault:            s.vault,
		Ratelimit:        s.ratelimit,
		OpenApiValidator: s.validator,
		Sender:           routes.NewJsonSender(s.logger),
	}

	s.logger.Info().Interface("svc", svc).Msg("Registering routes")

	staticBearerAuth := newBearerAuthMiddleware(s.authToken)

	v1Liveness.New(svc).Register(s.mux)
	openapi.New(svc).Register(s.mux)

	v1RatelimitCommitLease.New(svc).
		WithMiddleware(staticBearerAuth).
		Register(s.mux)

	v1RatelimitMultiRatelimit.New(svc).
		WithMiddleware(staticBearerAuth).
		Register(s.mux)

	v1RatelimitRatelimit.New(svc).
		WithMiddleware(staticBearerAuth).
		Register(s.mux)

	v1VaultDecrypt.New(svc).
		WithMiddleware(staticBearerAuth).
		Register(s.mux)

	v1VaultEncrypt.New(svc).
		WithMiddleware(staticBearerAuth).
		Register(s.mux)

	v1VaultEncryptBulk.New(svc).
		WithMiddleware(staticBearerAuth).
		Register(s.mux)

	notFound.New(svc).Register(s.mux)
}
