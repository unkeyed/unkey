package routes

import (
	v2EcsMeta "github.com/unkeyed/unkey/go/cmd/api/routes/v2_ecs_meta"
	v2Liveness "github.com/unkeyed/unkey/go/cmd/api/routes/v2_liveness"
	v2RatelimitLimit "github.com/unkeyed/unkey/go/cmd/api/routes/v2_ratelimit_limit"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
)

// here we register all of the routes.
// this function runs during startup.
func Register(srv *zen.Server, svc *Services) {
	withMetrics := zen.WithMetrics(svc.EventBuffer)
	withRootKeyAuth := zen.WithRootKeyAuth(svc.Keys)
	withLogging := zen.WithLogging(svc.Logger)
	withErrorHandling := zen.WithErrorHandling()
	withValidation := zen.WithValidation(svc.Validator)

	defaultMiddlewares := []zen.Middleware{
		withMetrics,
		withLogging,
		withErrorHandling,
		withRootKeyAuth, // must be before validation to capture the workspaceID
		withValidation,
	}

	srv.RegisterRoute(
		[]zen.Middleware{
			withMetrics,
			withLogging,
			withErrorHandling,
			withValidation,
		},
		v2Liveness.New())

	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitLimit.New(v2RatelimitLimit.Services{}),
	)

	srv.RegisterRoute([]zen.Middleware{}, v2EcsMeta.New())

}
