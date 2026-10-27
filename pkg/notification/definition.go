package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// Content is the channel-independent rendering of an event; HTML is optional.
type Content struct {
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html,omitempty"`
}

// Definition renders one event type independently of its delivery targets.
type Definition interface {
	// Type identifies the event type accepted by Render.
	Type() string
	// Render validates the event's schema and returns its channel-independent content.
	Render(Event) (Content, error)
}

// JSONDefinition decodes one event type and schema version into T.
// RenderPayload is required for Render, but not for Decode.
type JSONDefinition[T any] struct {
	Name          string
	SchemaVersion uint32
	RenderPayload func(T) (Content, error)
}

// Type returns the event type matched by Decode and Render.
func (d JSONDefinition[T]) Type() string { return d.Name }

// Render decodes the payload before invoking RenderPayload.
// Decode failures return zero Content; renderer results are returned unchanged.
func (d JSONDefinition[T]) Render(event Event) (Content, error) {
	if d.RenderPayload == nil {
		return Content{}, errors.New("unsupported notification definition or version")
	}
	payload, err := d.Decode(event)
	if err != nil {
		return Content{}, err
	}
	return d.RenderPayload(payload)
}

// Decode checks the type, version, and canonical payload, rejecting unknown fields.
// It does not validate the rest of the envelope and may return a partial T on error.
func (d JSONDefinition[T]) Decode(event Event) (T, error) {
	var payload T
	if event.Type != d.Name || event.SchemaVersion != d.SchemaVersion {
		return payload, errors.New("unsupported notification definition or version")
	}
	payloadJSON, err := normalizePayload(event.Payload)
	if err != nil {
		return payload, errors.New("invalid notification payload")
	}
	decoder := json.NewDecoder(bytes.NewReader(payloadJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return payload, errors.New("invalid notification payload")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return payload, errors.New("invalid notification payload")
	}
	return payload, nil
}
