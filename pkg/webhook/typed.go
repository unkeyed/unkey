package webhook

import (
	"context"
	"encoding/json"
	"fmt"
)

// On registers a handler for the given event types, replacing previous handlers.
// It decodes the payload as T; invalid JSON returns ErrBadRequest.
func (rec *Receiver) On[T any](eventTypes []string, fn func(ctx context.Context, event Event, payload T) error) *Receiver {
	return rec.OnRaw(eventTypes, func(ctx context.Context, event Event) error {
		payload, err := decodePayload[T](event)
		if err != nil {
			return err
		}
		return fn(ctx, event, payload)
	})
}

// OnResponse registers a handler with a JSON acknowledgment and the given status.
// Invalid JSON returns ErrBadRequest; handler or encoding errors discard the response.
func (rec *Receiver) OnResponse[T, R any](eventTypes []string, statusCode int, fn func(ctx context.Context, event Event, payload T) (R, error)) *Receiver {
	return rec.onResponse(eventTypes, func(ctx context.Context, event Event) (Response, error) {
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
	})
}

func decodePayload[T any](event Event) (T, error) {
	var payload T
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return payload, fmt.Errorf("%w: parse %s payload: %v", ErrBadRequest, event.Type, err)
	}
	return payload, nil
}
