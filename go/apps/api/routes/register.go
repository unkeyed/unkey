package routes

import (
	v2Liveness "github.com/unkeyed/unkey/go/apps/api/routes/v2_liveness"
	v2RatelimitDeleteOverride "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_delete_override"
	v2RatelimitGetOverride "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_get_override"
	v2RatelimitLimit "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	v2RatelimitSetOverride "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_set_override"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
)

// here we register all of the routes.
// this function runs during startup.
func Register(srv *zen.Server, svc *Services) {
	withTracing := zen.WithTracing()
	withMetrics := zen.WithMetrics(svc.EventBuffer)

	withLogging := zen.WithLogging(svc.Logger)
	withErrorHandling := zen.WithErrorHandling(svc.Logger)
	withValidation := zen.WithValidation(svc.Validator)

	defaultMiddlewares := []zen.Middleware{
		withTracing,
		withMetrics,
		withLogging,
		withErrorHandling,
		withValidation,
	}

	srv.RegisterRoute(
		[]zen.Middleware{
			withTracing,
			withMetrics,
			withLogging,
			withErrorHandling,
			withValidation,
		},
		v2Liveness.New())

	// ---------------------------------------------------------------------------
	// v2/ratelimit

	// v2/ratelimit.limit
	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitLimit.New(v2RatelimitLimit.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Ratelimit:   svc.Ratelimit,
			Permissions: svc.Permissions,
		}),
	)
	// v2/ratelimit.setOverride
	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitSetOverride.New(v2RatelimitSetOverride.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitGetOverride.New(v2RatelimitGetOverride.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitDeleteOverride.New(v2RatelimitDeleteOverride.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// ---------------------------------------------------------------------------
	// misc

}
