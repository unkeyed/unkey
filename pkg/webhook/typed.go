package webhook

import (
	"context"
	"encoding/json"
	"fmt"
)

// Typed adapts a handler that takes a parsed payload into a [HandlerFunc],
// removing the per-handler unmarshal boilerplate:
//
//	receiver.On([]string{"push"}, webhook.Typed(s.handlePush))
//
//	func (s *service) handlePush(ctx context.Context, event webhook.Event, payload pushPayload) error
//
// The event's raw payload is unmarshalled into T before the handler runs. A
// payload that does not parse maps to HTTP 400, outcome "bad_request" (via
// [ErrBadRequest]).
func Typed[T any](fn func(ctx context.Context, event Event, payload T) error) HandlerFunc {
	return func(ctx context.Context, event Event) error {
		payload, err := decodePayload[T](event)
		if err != nil {
			return err
		}
		return fn(ctx, event, payload)
	}
}

// TypedResponse decodes the payload as T and encodes the handler's result as JSON
// with the given status code. Invalid JSON returns ErrBadRequest. Handler and
// encoding errors use the receiver's normal error mapping; no success body is sent.
func TypedResponse[T, R any](
	statusCode int,
	fn func(ctx context.Context, event Event, payload T) (R, error),
) ResponseHandlerFunc {
	return func(ctx context.Context, event Event) (Response, error) {
		payload, err := decodePayload[T](event)
		if err != nil {
			return Response{}, err
		}
		result, err := fn(ctx, event, payload)
		if err != nil {
			return Response{}, err
		}

		body, err := json.Marshal(result)
		if err != nil {
			return Response{}, fmt.Errorf("encode %s response: %w", event.Type, err)
		}
		return Response{StatusCode: statusCode, ContentType: "application/json", Body: body}, nil
	}
}

func decodePayload[T any](event Event) (T, error) {
	var payload T
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return payload, fmt.Errorf("%w: parse %s payload: %v", ErrBadRequest, event.Type, err)
	}
	return payload, nil
}
