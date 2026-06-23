// Package server exposes the HTTP endpoints: a health check and the GitHub
// webhook receiver.
package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/go-github/v66/github"

	"github.com/unkeyed/unkey/tools/pr-channels/internal/app"
	"github.com/unkeyed/unkey/tools/pr-channels/internal/store"
)

// Server handles incoming webhooks.
type Server struct {
	app    *app.App
	store  *store.Store
	secret []byte
	log    *slog.Logger
}

// New builds a Server.
func New(a *app.App, st *store.Store, webhookSecret string, log *slog.Logger) *Server {
	return &Server{app: a, store: st, secret: []byte(webhookSecret), log: log}
}

// Handler returns the HTTP router.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("POST /webhook/github", s.handleGitHub)
	return mux
}

func (s *Server) handleGitHub(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, s.secret)
	if err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Idempotency: GitHub retries deliveries; act on each id at most once.
	deliveryID := github.DeliveryID(r)
	seen, err := s.store.SeenDelivery(r.Context(), deliveryID)
	if err != nil {
		s.log.Error("delivery dedup failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if seen {
		w.WriteHeader(http.StatusOK)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		http.Error(w, "cannot parse event", http.StatusBadRequest)
		return
	}

	// Acknowledge fast, process in the background. GitHub expects a quick 2xx;
	// Slack calls can take a moment under rate limiting.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.app.Handle(ctx, event); err != nil && !errors.Is(err, context.Canceled) {
			s.log.Error("handle event failed",
				"type", github.WebHookType(r), "delivery", deliveryID, "err", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}
