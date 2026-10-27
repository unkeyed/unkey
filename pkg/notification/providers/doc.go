// Package providers implements single-attempt notification delivery over HTTP.
//
// Constructors return the shared notification.Provider contract. Each instance
// uses an SSRF-protected client with a ten-second timeout. Sends do not retry;
// callers use notification.SendError and the provider's idempotency window to
// decide whether a retry is safe. Errors omit credentials and response bodies.
//
// # Usage
//
// Given a resolved notification.Request:
//
//	provider := providers.NewWebhook()
//	receipt, err := provider.Send(ctx, request)
//	if err != nil {
//		return err
//	}
//
// A successful receipt means provider acceptance, not end-recipient delivery.
package providers
