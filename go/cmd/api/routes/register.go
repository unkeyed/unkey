package routes

import (
	v2RatelimitLimit "github.com/unkeyed/unkey/go/cmd/api/routes/v2_ratelimit_limit"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
)

// here we register all of the routes.
// this function runs during startup.
func Register(srv *zen.Server, svc *Services) {
	srv.RegisterRoute(v2RatelimitLimit.New(v2RatelimitLimit.Services{}))
}
