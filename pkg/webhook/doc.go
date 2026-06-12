// Package webhook receives provider webhooks over HTTP.
//
// Every inbound webhook integration repeats the same transport chores:
// method and body-size policing, signature verification, routing by event
// type, deciding which failures the provider should retry, and counting all
// of it so nobody flies blind. This package owns those chores once, so an
// integration only writes event handlers:
//
//	receiver := webhook.New("stripe", stripeverifier.New(secret)). // pkg/webhook/verifiers/stripe
//		On(handleInvoiceCreated, "invoice.created").
//		On(handlePaymentFailed, "invoice.payment_failed")
//	mux.Handle("POST /webhooks/stripe", receiver)
//
// The handler comes first in [Receiver.On] so one handler can serve several
// event types: On(handleBranchLifecycle, "create", "delete"). Handlers that
// parse the payload wrap themselves in [Typed] to receive it already
// unmarshalled: On(webhook.Typed(handleInvoiceCreated), "invoice.created").
//
// Handler semantics drive the HTTP response and the metrics outcome:
//
//   - return nil: handled, 200. The provider considers the event delivered.
//   - return an error wrapping [ErrIgnore]: ignored, 200. The handler looked
//     at the event and deliberately declined it (wrong subtype, not our
//     tenant). Acknowledged so the provider does not retry, but counted
//     separately from handled.
//   - return any other error: error, 500. The provider retries, so handlers
//     must be idempotent.
//
// Events without a registered handler are acknowledged with 200 and counted
// as unhandled; providers treat non-2xx as "retry forever", so refusing
// uninteresting event types would only create noise on both sides. Register
// a [Receiver.Default] handler to take over that fallback path instead.
//
// # Durability
//
// The receiver is plain HTTP on purpose; it does not wrap handlers in
// Restate. Handlers should do their cheap synchronous filtering inline
// (parse the payload, decide relevance) and then dispatch the actual work
// into a Restate handler via the ingress client, ideally with an idempotency
// key — see ctrl-api's Stripe and GitHub webhooks for the pattern. Forcing
// every event through Restate at the transport layer would journal ignored
// events and drag virtual-object key derivation into code that cannot know
// it.
package webhook
