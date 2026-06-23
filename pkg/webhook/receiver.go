package webhook

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
)

// defaultMaxBodySize bounds webhook payloads when the caller doesn't override
// it. Providers send events, not uploads; 2 MB is far above any real event
// while still bounding memory per request. Providers with larger caps (GitHub
// allows up to 25 MB) can raise it with [WithMaxBodySize].
const defaultMaxBodySize int64 = 2 * 1024 * 1024

// Receiver is an http.Handler that verifies and routes provider webhooks.
//
// Register handlers and middleware (On, Default, Use) during setup, before the
// Receiver is served. Once ServeHTTP has been called it is read-only and safe
// for concurrent requests; registering on a live Receiver races, like writing
// to an http.ServeMux after it starts serving.
type Receiver struct {
	provider    string
	verifier    Verifier
	handlers    map[string]HandlerFunc
	fallback    HandlerFunc
	middlewares []Middleware
	maxBodySize int64
}

var _ http.Handler = (*Receiver)(nil)

// Option configures a Receiver at construction.
type Option func(*Receiver)

// WithMaxBodySize overrides the maximum accepted request body, in bytes. The
// default is 2 MB; providers with larger payloads (GitHub caps at 25 MB) can
// raise it to match their limit.
func WithMaxBodySize(n int64) Option {
	return func(rec *Receiver) { rec.maxBodySize = n }
}

// New builds a Receiver for one provider endpoint. The provider name labels
// metrics and logs.
func New(provider string, verifier Verifier, opts ...Option) *Receiver {
	rec := &Receiver{
		provider:    provider,
		verifier:    verifier,
		handlers:    map[string]HandlerFunc{},
		fallback:    nil,
		middlewares: nil,
		maxBodySize: defaultMaxBodySize,
	}
	for _, opt := range opts {
		opt(rec)
	}
	return rec
}

// On registers the handler for the given event types, replacing any previous
// registration. Reads as "on these events, run this handler". Returns the
// Receiver for chaining.
func (rec *Receiver) On(eventTypes []string, fn HandlerFunc) *Receiver {
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

	// Bound the body before the verifier reads it; providers send events, not
	// uploads. The verifier owns the single read, so the raw bytes live in one
	// place rather than being read here and handed down too.
	r.Body = http.MaxBytesReader(w, r.Body, rec.maxBodySize)

	event, err := rec.verifier.Verify(r)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			rec.count("none", "too_large")
			logger.Warn("webhook rejected: body too large", "provider", rec.provider)
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
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
	case errors.Is(err, ErrBadRequest):
		rec.count(event.Type, "bad_request")
		logger.Warn("webhook bad request",
			"provider", rec.provider,
			"event", event.Type,
			"event_id", event.ID,
			"error", err,
		)
		// 4xx: the verified request is malformed, so retrying the same bytes
		// would fail identically. The provider should not keep retrying.
		http.Error(w, fmt.Sprintf("handling %s failed: bad request", event.Type), http.StatusBadRequest)
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
