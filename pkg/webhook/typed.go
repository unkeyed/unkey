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
// payload that does not parse maps to HTTP 400, outcome "bad_request" (via
// [ErrBadRequest]): the request was signature-verified, so the same bytes will
// fail to parse on every retry, and a 5xx would have the provider hammer a
// poison payload. T should parse minimally, reading only the fields the
// handler needs.
func Typed[T any](fn func(ctx context.Context, event Event, payload T) error) HandlerFunc {
	return func(ctx context.Context, event Event) error {
		var payload T
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("%w: parse %s payload: %v", ErrBadRequest, event.Type, err)
		}
		return fn(ctx, event, payload)
	}
}
