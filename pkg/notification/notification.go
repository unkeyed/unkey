package notification

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type Channel string

const (
	Email   Channel = "email"
	Slack   Channel = "slack"
	Discord Channel = "discord"
	Webhook Channel = "webhook"
)

type Resource struct {
	ProjectID     string `json:"projectId,omitempty"`
	AppID         string `json:"appId,omitempty"`
	EnvironmentID string `json:"environmentId,omitempty"`
}

// Event identifies one occurrence. Retries must keep both its identity and contents unchanged.
type Event struct {
	WorkspaceID   string          `json:"workspaceId"`
	SourceEventID string          `json:"sourceEventId"`
	Type          string          `json:"type"`
	SchemaVersion uint32          `json:"schemaVersion"`
	Resource      Resource        `json:"resource"`
	Payload       json.RawMessage `json:"payload"`
	OccurredAt    int64           `json:"occurredAt"`
	ExpiresAt     int64           `json:"expiresAt,omitempty"`
}

type Content struct {
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html,omitempty"`
}

type Definition interface {
	Type() string
	Render(Event) (Content, error)
}

// Notifier acknowledges durable enqueue, not provider acceptance or delivery.
type Notifier interface {
	Notify(context.Context, Event) (string, error)
}

// Target contains plaintext credentials. Never persist or journal it without encryption.
type Target struct {
	Email         string            `json:"email,omitempty"`
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	SigningSecret string            `json:"signingSecret,omitempty"`
}

type Request struct {
	DeliveryID string  `json:"deliveryId"`
	Event      Event   `json:"event"`
	Content    Content `json:"content"`
	Target     Target  `json:"target"`
}

type Receipt struct {
	MessageID string `json:"messageId"`
}

// Provider sends once, without internal retries. IdempotencyWindow is the provider's
// guaranteed retention of DeliveryID; zero means ambiguous sends cannot be retried safely.
type Provider interface {
	Name() string
	Channel() Channel
	IdempotencyWindow() time.Duration
	Send(context.Context, Request) (Receipt, error)
}

type FailureKind string

const (
	Rejected  FailureKind = "rejected"
	Retryable FailureKind = "retryable"
	Ambiguous FailureKind = "ambiguous"
)

// SendError must not contain recipient addresses, webhook URLs, credentials, or response bodies.
// Retryable means the request was not sent or the provider explicitly did not accept it.
// Unknown outcomes are Ambiguous.
type SendError struct {
	Kind       FailureKind
	Code       string
	RetryAfter time.Duration
}

func (e *SendError) Error() string { return e.Code }

func (e Event) Validate() error {
	_, err := e.Normalize()
	return err
}

func (e Event) Normalize() (Event, error) {
	type boundedField struct {
		name     string
		value    string
		bytesMax int
	}
	for _, field := range []boundedField{
		{name: "workspaceId", value: e.WorkspaceID, bytesMax: 48},
		{name: "sourceEventId", value: e.SourceEventID, bytesMax: 256},
		{name: "type", value: e.Type, bytesMax: 128},
	} {
		if len(field.value) == 0 || len(field.value) > field.bytesMax || strings.TrimSpace(field.value) != field.value {
			return Event{}, fmt.Errorf("invalid %s", field.name)
		}
	}
	if e.SchemaVersion == 0 || e.OccurredAt <= 0 || (e.ExpiresAt != 0 && e.ExpiresAt <= e.OccurredAt) {
		return Event{}, errors.New("invalid event version or timestamps")
	}
	if len(e.Resource.ProjectID) > 48 || len(e.Resource.AppID) > 48 || len(e.Resource.EnvironmentID) > 48 {
		return Event{}, errors.New("invalid resource identifiers")
	}
	if (e.Resource.AppID != "" && e.Resource.ProjectID == "") || (e.Resource.EnvironmentID != "" && e.Resource.AppID == "") {
		return Event{}, errors.New("resource scope requires its parents")
	}
	payload, err := normalizePayload(e.Payload)
	if err != nil {
		return Event{}, err
	}
	e.Payload = payload
	return e, nil
}

func (e Event) ID() string {
	return stableID("ntf", e.WorkspaceID, e.Type, e.SourceEventID)
}

func DeliveryID(notificationID string, channel Channel, targetKey string) string {
	return stableID("ndlv", notificationID, string(channel), targetKey)
}

func stableID(prefix string, parts ...string) string {
	var encoded []byte
	for _, part := range parts {
		encoded = fmt.Appendf(encoded, "%d:%s", len(part), part)
	}
	sum := sha256.Sum256(encoded)
	return prefix + "_" + hex.EncodeToString(sum[:20])
}

func (e Event) Fingerprint() (string, error) {
	e, err := e.Normalize()
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

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
