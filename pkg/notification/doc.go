// Package notification defines durable event enqueueing and single-attempt delivery contracts.
//
// Callers keep an Event's source identity and contents stable across retries.
// [Event.ID] identifies the occurrence; [Event.Fingerprint] detects changed contents.
// [Definition] renders events, while [Provider] sends to resolved targets.
// Targets contain secrets and must not be persisted or journaled in plaintext.
//
// # Usage
//
// Use [NewClient] with Restate ingress outside a handler, or [Send] inside a
// handler and outside restate.Run. Both validate events before enqueueing.
// Given an initialized client and event:
//
//	id, err := client.Notify(ctx, event)
//	if err != nil {
//		return err
//	}
//
// The returned ID acknowledges durable enqueueing, not provider delivery.
package notification
