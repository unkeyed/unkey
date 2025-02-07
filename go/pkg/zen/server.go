package zen

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/certificate"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type Server struct {
	mu sync.Mutex

	logger      logging.Logger
	isListening bool
	mux         *http.ServeMux
	srv         *http.Server

	sessions sync.Pool
}

type Config struct {
	NodeID string
	Logger logging.Logger

	CertificateSource certificate.Source
}

func New(config Config) (*Server, error) {
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler: mux,
		// See https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		//
		// > # http.ListenAndServe is doing it wrong
		// > Incidentally, this means that the package-level convenience functions that bypass http.Server
		// > like http.ListenAndServe, http.ListenAndServeTLS and http.Serve are unfit for public Internet
		// > Servers.
		// >
		// > Those functions leave the Timeouts to their default off value, with no way of enabling them,
		// > so if you use them you'll soon be leaking connections and run out of file descriptors. I've
		// > made this mistake at least half a dozen times.
		// >
		// > Instead, create a http.Server instance with ReadTimeout and WriteTimeout and use its
		// > corresponding methods, like in the example a few paragraphs above.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	if config.CertificateSource != nil {
		config.Logger.Info(context.Background(), "using certificate source")
		srv.TLSConfig = &tls.Config{
			// which are tuned to avoid attacks. Does nothing on clients.
			PreferServerCipherSuites: true,
			// Only use curves which have assembly implementations
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},

			GetCertificate: config.CertificateSource.GetCertificate,
		}
	}

	s := &Server{
		mu:          sync.Mutex{},
		logger:      config.Logger,
		isListening: false,
		mux:         mux,
		srv:         srv,
		sessions: sync.Pool{
			New: func() any {
				return &Session{
					requestID:      "",
					w:              nil,
					r:              nil,
					requestBody:    []byte{},
					responseStatus: 0,
					responseBody:   []byte{},
				}
			},
		},
	}
	return s, nil
}

// Get a fresh or reused session from the pool.
//
// You must return it to the pool when you are done via s.returnSession()
//
// Returning proper types is a bit annoying here, cause each session might have
// different requets and response types, so we just return any.
// You should immediately cast it to your desired type
//
// sess := s.getSession().(session.Session[MyRequest, MyResponse]).
func (s *Server) getSession() any {
	return s.sessions.Get()
}

// Return the session to the sync pool.
func (s *Server) returnSession(session any) {
	s.sessions.Put(session)
}

// Mux returns the underlying http.ServeMux.
//
// Usually you don't need to use this, but it's here for tests.
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

// Calling this function multiple times will have no effect.
func (s *Server) Listen(ctx context.Context, addr string) error {
	s.mu.Lock()
	if s.isListening {
		s.logger.Warn(ctx, "already listening")
		s.mu.Unlock()
		return nil
	}
	s.isListening = true
	s.mu.Unlock()

	s.srv.Addr = addr

	s.logger.Info(ctx, "listening", slog.String("addr", addr))

	ln, err := tls.Listen("tcp", addr, s.srv.TLSConfig)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("creating listener failed", ""))
	}
	err = http.Serve(ln, s.srv.Handler)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("listening failed", ""))
	}
	return nil
}

func (s *Server) RegisterRoute(middlewares []Middleware, route Route) {
	s.mux.HandleFunc(
		fmt.Sprintf("%s %s", route.Method(), route.Path()),
		func(w http.ResponseWriter, r *http.Request) {
			sess, ok := s.getSession().(*Session)
			if !ok {
				panic("Unable to cast session")
			}
			defer func() {
				sess.reset()
				s.returnSession(sess)
			}()

			err := sess.Init(w, r)
			if err != nil {
				s.logger.Error(context.Background(), "failed to init session")
				return
			}

			// Apply middleware
			var handle HandleFunc = route.Handle

			// Reverses the middlewares to run in the desired order.
			// If middlewares are [A, B, C], this writes [C, B, A] to s.middlewares.
			for i := len(middlewares) - 1; i >= 0; i-- {
				handle = middlewares[i](handle)
			}

			err = handle(sess)

			if err != nil {
				panic(err)
			}
		})
}

func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.srv.Close()
	if err != nil {
		return fault.Wrap(err)
	}
	return nil
}
