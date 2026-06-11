package webhook

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
)

// maxBodySize bounds webhook payloads. Providers send events, not uploads;
// 2 MB is far above any real event while still bounding memory per request.
const maxBodySize = 2 * 1024 * 1024

// Receiver is an http.Handler that verifies and routes provider webhooks.
type Receiver struct {
	provider    string
	verifier    Verifier
	handlers    map[string]HandlerFunc
	fallback    HandlerFunc
	middlewares []Middleware
}

var _ http.Handler = (*Receiver)(nil)

// New builds a Receiver for one provider endpoint. The provider name labels
// metrics and logs.
func New(provider string, verifier Verifier) *Receiver {
	return &Receiver{
		provider:    provider,
		verifier:    verifier,
		handlers:    map[string]HandlerFunc{},
		fallback:    nil,
		middlewares: nil,
	}
}

// On registers the handler for one or more event types, replacing any
// previous registration. The handler comes first because the event types are
// variadic. Returns the Receiver for chaining.
func (rec *Receiver) On(fn HandlerFunc, eventTypes ...string) *Receiver {
	if len(eventTypes) == 0 {
		panic("webhook: On requires at least one event type")
	}
	for _, eventType := range eventTypes {
		rec.handlers[eventType] = fn
	}
	return rec
}

// Default registers the fallback handler for event types with no [On]
// registration. Without one, unknown events are acknowledged with 200 and
// counted as unhandled; with one, the handler decides the outcome like any
// other (return [ErrIgnore] to keep the 200-and-count-separately behavior
// with a logged reason). Returns the Receiver for chaining.
func (rec *Receiver) Default(fn HandlerFunc) *Receiver {
	rec.fallback = fn
	return rec
}

// Use appends middlewares that wrap every handler, applied outermost first
// (the first Use wraps closest to the transport). Returns the Receiver for
// chaining.
func (rec *Receiver) Use(mw ...Middleware) *Receiver {
	rec.middlewares = append(rec.middlewares, mw...)
	return rec
}

func (rec *Receiver) count(event, outcome string) {
	eventsTotal.WithLabelValues(rec.provider, event, outcome).Inc()
}

func (rec *Receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodySize))
	if err != nil {
		rec.count("none", "bad_request")
		logger.Warn("webhook rejected: failed to read body", "provider", rec.provider, "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	event, err := rec.verifier.Verify(r, body)
	if err != nil {
		rec.count("none", "verification_failed")
		logger.Warn("webhook rejected: verification failed", "provider", rec.provider, "error", err)
		http.Error(w, "verification failed", http.StatusUnauthorized)
		return
	}

	handler, ok := rec.handlers[event.Type]
	if !ok {
		if rec.fallback == nil {
			rec.count(event.Type, "unhandled")
			w.WriteHeader(http.StatusOK)
			return
		}
		handler = rec.fallback
	}

	for i := len(rec.middlewares) - 1; i >= 0; i-- {
		handler = rec.middlewares[i](handler)
	}

	start := time.Now()
	err = handler(r.Context(), event)
	handlerDuration.WithLabelValues(rec.provider, event.Type).
		Observe(time.Since(start).Seconds())

	switch {
	case err == nil:
		rec.count(event.Type, "handled")
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, ErrIgnore):
		rec.count(event.Type, "ignored")
		logger.Info("webhook event ignored",
			"provider", rec.provider,
			"event", event.Type,
			"event_id", event.ID,
			"reason", err.Error(),
		)
		w.WriteHeader(http.StatusOK)
	default:
		rec.count(event.Type, "error")
		logger.Error("webhook handler failed",
			"provider", rec.provider,
			"event", event.Type,
			"event_id", event.ID,
			"error", err,
		)
		// 5xx so the provider retries; handlers are required to be idempotent.
		http.Error(w, fmt.Sprintf("handling %s failed", event.Type), http.StatusInternalServerError)
	}
}
