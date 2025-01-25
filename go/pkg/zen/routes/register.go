package routes

import (
	zen "github.com/unkeyed/unkey/go/pkg/zen"
	v2RatelimitLimit "github.com/unkeyed/unkey/go/pkg/zen/routes/v2_ratelimit_limit"
)

// here we register all of the routes.
// this function runs during startup.
func Register(srv *zen.Server, svc *zen.Services) {
	srv.RegisterRoute(v2RatelimitLimit.New(svc))
}
