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
	"github.com/unkeyed/unkey/svc/kitchensink/logs"
	"github.com/unkeyed/unkey/svc/kitchensink/principal"
	"github.com/unkeyed/unkey/svc/kitchensink/sleep"
	"github.com/unkeyed/unkey/svc/kitchensink/status"
)

// routes is the explicit registry of method+path → handler. To add a
// new route, create a subpackage under svc/kitchensink/ that exports
// `func Handler(w http.ResponseWriter, r *http.Request)` and append
// one line here. See README.md.
var routes = map[string]http.HandlerFunc{
	"GET /hello":         hello.Handler,
	"GET /env":           env.Handler,
	"GET /principal":     principal.Handler,
	"GET /headers":       headers.Handler,
	"POST /echo":         echo.Handler,
	"POST /log":          logs.Handler,
	"GET /status/{code}": status.Handler,
	"GET /sleep":         sleep.Handler,
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
	for pattern, fn := range routes {
		mux.HandleFunc(pattern, fn)
		logger.Info("registered", "pattern", pattern)
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
