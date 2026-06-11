package webhook

import (
	"context"
	"encoding/json"
	"fmt"
)

// Typed adapts a handler that takes a parsed payload into a [HandlerFunc],
// removing the per-handler unmarshal boilerplate:
//
//	receiver.On(webhook.Typed(s.handlePush), "push")
//
//	func (s *service) handlePush(ctx context.Context, event webhook.Event, payload pushPayload) error
//
// The event's raw payload is unmarshalled into T before the handler runs. A
// payload that does not parse is returned as a handler error (HTTP 500,
// outcome "error"): the request was signature-verified, so a malformed
// payload means a contract drift worth retrying and alerting on, not an
// event to acknowledge. T should parse minimally, reading only the fields
// the handler needs.
func Typed[T any](fn func(ctx context.Context, event Event, payload T) error) HandlerFunc {
	return func(ctx context.Context, event Event) error {
		var payload T
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("parse %s payload: %w", event.Type, err)
		}
		return fn(ctx, event, payload)
	}
}
