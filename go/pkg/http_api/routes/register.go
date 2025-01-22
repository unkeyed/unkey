package routes

import (
	httpApi "github.com/unkeyed/unkey/go/pkg/http_api"
	v1RatelimitLimit "github.com/unkeyed/unkey/go/pkg/http_api/routes/v1_ratelimit_limit"
)

// here we register all of the routes
// this function runs during startup
func Register(srv *httpApi.Server, svc *httpApi.Services) {

	register(srv, v1RatelimitLimit.New(svc))
}

// register is a hack to go from a parameterized generic function to the base
// interface.
// Either go or I am not smart enough to figure out how to extend generic
// interfaces and make it work.
//
// This only happens during startup of a node, so even if it panics, that's ok
func register[TRequest httpApi.Redacter, TResponse httpApi.Redacter](srv *httpApi.Server, route httpApi.Route[TRequest, TResponse]) {

	srv.RegisterRoute(route.(httpApi.Route[httpApi.Redacter, httpApi.Redacter]))
}
