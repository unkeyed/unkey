package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func normalizePayload(payload json.RawMessage) (json.RawMessage, error) {
	if len(payload) > 64*1024 || !json.Valid(payload) {
		return nil, errors.New("payload must be a JSON object of at most 64 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	value, err := decodeJSONValue(decoder)
	if err != nil {
		return nil, errors.New("payload must be unambiguous JSON")
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, errors.New("payload must be a JSON object of at most 64 KiB")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, errors.New("payload must be unambiguous JSON")
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical payload: %w", err)
	}
	if len(canonical) > 64*1024 {
		return nil, errors.New("payload must be a JSON object of at most 64 KiB")
	}
	return canonical, nil
}

func decodeJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delimiter {
	case '{':
		object := make(map[string]any)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, errors.New("object key is not a string")
			}
			if _, exists := object[key]; exists {
				return nil, fmt.Errorf("duplicate object key %q", key)
			}
			object[key], err = decodeJSONValue(decoder)
			if err != nil {
				return nil, err
			}
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return object, nil
	case '[':
		array := make([]any, 0)
		for decoder.More() {
			value, err := decodeJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return array, nil
	default:
		return nil, errors.New("unexpected JSON delimiter")
	}
}
