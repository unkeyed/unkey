package connect

import (
	"context"
	"fmt"

	"net/http"
	"sync"

	"github.com/bufbuild/connect-go"
	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	vaultv1connect "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/apps/vault/pkg/logging"
	"github.com/unkeyed/unkey/apps/vault/pkg/service"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	sync.RWMutex
	isListening    bool
	isShuttingDown bool
	logger         logging.Logger
	shutdownC      chan struct{}
	svc            *service.Service

	vaultv1connect.UnimplementedVaultServiceHandler // returns errors from all methods
}

type Config struct {
	Logger logging.Logger

	Service *service.Service
}

func New(cfg Config) (*Server, error) {

	return &Server{
		logger:         cfg.Logger,
		isListening:    false,
		isShuttingDown: false,
		svc:            cfg.Service,

		shutdownC: make(chan struct{}),
	}, nil
}

func (s *Server) Decrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.DecryptRequest],
) (*connect.Response[vaultv1.DecryptResponse], error) {
	res, err := s.svc.Decrypt(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return connect.NewResponse(res), nil

}
func (s *Server) Encrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.EncryptRequest],
) (*connect.Response[vaultv1.EncryptResponse], error) {
	res, err := s.svc.Encrypt(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *Server) CreateDEK(ctx context.Context, req *connect.Request[vaultv1.CreateDEKRequest]) (*connect.Response[vaultv1.CreateDEKResponse], error) {
	res, err := s.svc.CreateDEK(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dek: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *Server) Listen(addr string) error {
	s.Lock()
	if s.isListening {
		s.logger.Info().Msg("already listening")
		s.Unlock()
		return nil
	}
	s.isListening = true
	s.Unlock()

	mux := http.NewServeMux() // `NewServeMux` is a function in the `net/http` package in Go that creates a new HTTP request multiplexer (ServeMux). A ServeMux is an HTTP request router that matches the URL of incoming requests against a list of registered patterns and calls the handler for the pattern that most closely matches the URL path. It essentially acts as a router for incoming HTTP requests, directing them to the appropriate handler based on the URL path.NewServeMux()

	mux.Handle(vaultv1connect.NewVaultServiceHandler(s))

	srv := &http.Server{Addr: addr, Handler: h2c.NewHandler(mux, &http2.Server{})}

	s.logger.Info().Str("addr", addr).Msg("listening")
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			s.logger.Error().Err(err).Msg("listen and serve failed")
		}
	}()

	<-s.shutdownC
	s.logger.Info().Msg("shutting down")
	return srv.Shutdown(context.Background())

}
