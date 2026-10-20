package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		logger.Error("deploy demo stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		return fault.New("PORT must be an integer between 1 and 65535")
	}
	srv, err := newServer()
	if err != nil {
		return fault.Wrap(err, fault.Internal("create demo server"))
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("", port))
	if err != nil {
		return fault.Wrap(err, fault.Internal("listen on demo port"))
	}
	return srv.Serve(ctx, listener)
}

type messageResponse struct {
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func newServer() (*zen.Server, error) {
	var config zen.Config
	config.MaxRequestBodySize = 1024
	srv, err := zen.New(config)
	if err != nil {
		return nil, err
	}
	middlewares := []zen.Middleware{zen.WithPanicRecovery(), withErrors}
	srv.RegisterRoutingErrors(middlewares)
	srv.RegisterRoute(middlewares, zen.NewRoute(http.MethodGet, "/{$}", func(_ context.Context, s *zen.Session) error {
		return s.JSON(http.StatusOK, struct {
			Message   string   `json:"message"`
			Endpoints []string `json:"endpoints"`
			Next      string   `json:"next"`
			Docs      string   `json:"docs"`
		}{
			Message:   "Your Unkey Deploy demo is running.",
			Endpoints: []string{"GET /", "GET /health", "GET /api/hello"},
			Next:      "Add a Key Auth policy matching /api/, select a keyspace, then redeploy. The API is public until you do.",
			Docs:      "https://www.unkey.com/docs/compute/gateway/api-key-auth",
		})
	}))
	srv.RegisterRoute(middlewares, zen.NewRoute(http.MethodGet, "/health", func(_ context.Context, s *zen.Session) error {
		return s.JSON(http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ok"})
	}))
	srv.RegisterRoute(middlewares, zen.NewRoute(http.MethodGet, "/api/hello", func(_ context.Context, s *zen.Session) error {
		return s.JSON(http.StatusOK, messageResponse{
			Message:   "Hello from Unkey Deploy.",
			RequestID: s.RequestID(),
		})
	}))
	return srv, nil
}

func withErrors(next zen.HandleFunc) zen.HandleFunc {
	return func(ctx context.Context, s *zen.Session) error {
		s.ResponseWriter().Header().Set("Cache-Control", "no-store")
		err := next(ctx, s)
		if err == nil {
			return nil
		}
		status := http.StatusInternalServerError
		var routingError *zen.RoutingError
		if errors.As(err, &routingError) {
			status = routingError.Status
		} else if code, ok := fault.GetCode(err); ok && code == codes.UserErrorsBadRequestRequestBodyTooLarge {
			status = http.StatusRequestEntityTooLarge
		}
		return s.JSON(status, messageResponse{
			Message:   http.StatusText(status),
			RequestID: s.RequestID(),
		})
	}
}
