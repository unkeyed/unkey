package connect

import (
	"context"
	"fmt"

	"net/http"
	"sync"

	"github.com/bufbuild/connect-go"
	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	vaultv1connect "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/apps/vault/pkg/auth"
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

func (s *Server) Liveness(ctx context.Context, req *connect.Request[vaultv1.LivenessRequest]) (*connect.Response[vaultv1.LivenessResponse], error) {
	return connect.NewResponse(&vaultv1.LivenessResponse{
		Status: "serving",
	}), nil
}

func (s *Server) Decrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.DecryptRequest],
) (*connect.Response[vaultv1.DecryptResponse], error) {
	err := auth.Authorize(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Decrypt(ctx, req.Msg)
	if err != nil {
		s.logger.Err(err).Msg("failed to decrypt")

		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return connect.NewResponse(res), nil

}
func (s *Server) Encrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.EncryptRequest],
) (*connect.Response[vaultv1.EncryptResponse], error) {
	err := auth.Authorize(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	res, err := s.svc.Encrypt(ctx, req.Msg)
	if err != nil {
		s.logger.Err(err).Msg("failed to encrypt")
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return connect.NewResponse(res), nil
}

func (s *Server) EncryptBulk(
	ctx context.Context,
	req *connect.Request[vaultv1.EncryptBulkRequest],
) (*connect.Response[vaultv1.EncryptBulkResponse], error) {
	err := auth.Authorize(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	res, err := s.svc.EncryptBulk(ctx, req.Msg)
	if err != nil {
		s.logger.Err(err).Msg("failed to encrypt bulk")
		return nil, fmt.Errorf("failed to encrypt bulk: %w", err)
	}
	return connect.NewResponse(res), nil
}

func (s *Server) CreateDEK(
	ctx context.Context,
	req *connect.Request[vaultv1.CreateDEKRequest],
) (*connect.Response[vaultv1.CreateDEKResponse], error) {
	err := auth.Authorize(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	res, err := s.svc.CreateDEK(ctx, req.Msg)
	if err != nil {
		s.logger.Err(err).Msg("failed to create dek")
		return nil, fmt.Errorf("failed to create dek: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *Server) ReEncrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.ReEncryptRequest],
) (*connect.Response[vaultv1.ReEncryptResponse], error) {
	err := auth.Authorize(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	res, err := s.svc.ReEncrypt(ctx, req.Msg)
	if err != nil {
		s.logger.Err(err).Msg("failed to reencrypt")
		return nil, fmt.Errorf("failed to reencrypt: %w", err)
	}
	return connect.NewResponse(res), nil

}

type h struct {
	logger logging.Logger
	next   http.Handler
}

func (h *h) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	m := make(map[string][]string)
	for k, v := range r.Header.Clone() {
		m[k] = v
	
	}

	h.logger.Info().Str("method", r.Method).Str("path", r.URL.Path).Str("RemoteAddr", r.RemoteAddr).Interface("m",m).Msg("request")
	h.next.ServeHTTP(w, r)

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
	mux.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to write response")
		}
	})

	srv := &http.Server{Addr: addr, Handler: h2c.NewHandler(&h{logger: s.logger, next: mux}, &http2.Server{})}

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
