// Command kitchensink runs a stdlib-only HTTP server that exposes every
// probe subpackage as a real HTTP endpoint. See README.md.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/svc/kitchensink/echo"
	"github.com/unkeyed/unkey/svc/kitchensink/env"
	"github.com/unkeyed/unkey/svc/kitchensink/headers"
	"github.com/unkeyed/unkey/svc/kitchensink/hello"
	"github.com/unkeyed/unkey/svc/kitchensink/index"
	"github.com/unkeyed/unkey/svc/kitchensink/logs"
	"github.com/unkeyed/unkey/svc/kitchensink/principal"
	"github.com/unkeyed/unkey/svc/kitchensink/sleep"
	"github.com/unkeyed/unkey/svc/kitchensink/status"
)

// route ties a ServeMux pattern to its handler and a human description
// shown on the index page.
type route struct {
	Pattern     string
	Description string
	Fn          http.HandlerFunc
}

// routes is the explicit registry of every route the server serves.
// To add one: create a subpackage under svc/kitchensink/ that exports
// `func Handler(w http.ResponseWriter, r *http.Request)` and append a
// line here. See README.md.
var routes = []route{
	{"GET /", "This page — lists every registered route.", index.Handler},
	{"GET /hello", "Smoke test — returns 'hello, world'.", hello.Handler},
	{"GET /env", "Process environment as JSON. Filter with ?prefix=.", env.Handler},
	{"GET /principal", "Decodes X-Unkey-Principal and returns it.", principal.Handler},
	{"GET /headers", "Incoming request headers as JSON.", headers.Handler},
	{"POST /echo", "Returns the request body verbatim.", echo.Handler},
	{"POST /log", "Logs the body at INFO, echoes it.", logs.Handler},
	{"GET /status/{code}", "Returns whatever HTTP status you ask for.", status.Handler},
	{"GET /sleep", "Sleeps ?d=<duration>, honors cancellation.", sleep.Handler},
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	mux := http.NewServeMux()
	for _, rt := range routes {
		mux.HandleFunc(rt.Pattern, rt.Fn)
		index.Register(rt.Pattern, rt.Description)
		logger.Info("registered", "pattern", rt.Pattern)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("kitchensink listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
