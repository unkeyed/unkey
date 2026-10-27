package notification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Resource scopes an event to a project, app, or environment; narrower scopes require their parents.
type Resource struct {
	ProjectID     string `json:"projectId,omitempty"`
	AppID         string `json:"appId,omitempty"`
	EnvironmentID string `json:"environmentId,omitempty"`
}

// Event identifies one occurrence. Retries must keep both its identity and contents unchanged.
// Timestamps are Unix milliseconds; a zero ExpiresAt means no expiry.
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

// Validate checks the envelope and payload without changing the event.
func (e Event) Validate() error {
	_, err := e.Normalize()
	return err
}

// Normalize validates the envelope and canonicalizes its JSON object payload.
// It rejects duplicate keys and payloads exceeding 64 KiB before or after
// canonicalization, preserves JSON numbers, and returns a zero Event on error.
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

// ID derives a stable identity from the workspace, type, and source event ID.
// It does not validate the event or include its payload.
func (e Event) ID() string {
	return stableID("ntf", e.WorkspaceID, e.Type, e.SourceEventID)
}

// Fingerprint hashes the entire normalized event to detect changed contents
// under the same ID. It returns an empty string on validation or encoding errors.
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

func stableID(prefix string, parts ...string) string {
	var encoded []byte
	for _, part := range parts {
		encoded = fmt.Appendf(encoded, "%d:%s", len(part), part)
	}
	sum := sha256.Sum256(encoded)
	return prefix + "_" + hex.EncodeToString(sum[:20])
}
