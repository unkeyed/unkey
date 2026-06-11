package webhook

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// The transport metrics live here, next to the code that emits them, because
// they are generic to every webhook integration: a silent provider, a
// signature mismatch after a secret rotation, or a handler that started
// erroring all show up the same way regardless of provider. Integration-
// specific metrics belong next to the integration's handlers, layered in via
// [Receiver.Use] middleware.
var (
	// eventsTotal counts inbound webhook events by outcome.
	//
	// Labels:
	//   - "provider": the webhook source, e.g. "stripe"
	//   - "event": the provider's event type, e.g. "invoice.created".
	//     Bounded by the provider's event catalog.
	//   - "outcome": one of
	//     "handled" (a handler ran and succeeded),
	//     "ignored" (a handler looked and declined via ErrIgnore),
	//     "unhandled" (no handler registered for the event type),
	//     "error" (a handler failed; the provider will retry),
	//     "verification_failed" (signature rejected),
	//     "bad_request" (unreadable request).
	eventsTotal = lazy.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "webhook",
		Name:      "events_total",
		Help:      "Inbound webhook events by provider, event type and outcome.",
	}, []string{"provider", "event", "outcome"})

	// handlerDuration measures how long webhook event handlers run.
	// Providers retry on slow responses (Stripe times out around 20s), so a
	// drifting p99 here is an early warning before events start double
	// delivering.
	//
	// Labels:
	//   - "provider": the webhook source, e.g. "stripe"
	//   - "event": the provider's event type
	handlerDuration = lazy.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey",
		Subsystem: "webhook",
		Name:      "handler_duration_seconds",
		Help:      "Duration of webhook event handlers.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"provider", "event"})
)
