package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

type JSONDefinition[T any] struct {
	Name          string
	SchemaVersion uint32
	RenderPayload func(T) (Content, error)
}

func (d JSONDefinition[T]) Type() string { return d.Name }

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
