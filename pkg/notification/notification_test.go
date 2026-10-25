package notification

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentityAndFingerprint(t *testing.T) {
	event := Event{WorkspaceID: "w1", Type: "deployment.failed", SourceEventID: "d1", SchemaVersion: 1, OccurredAt: 1, Payload: json.RawMessage(`{"a":9007199254740993,"b":2}`)}
	hash, err := event.Fingerprint()
	require.NoError(t, err)
	id := event.ID()
	event.Payload = json.RawMessage(`{ "b": 2, "a": 9007199254740993 }`)
	same, err := event.Fingerprint()
	require.NoError(t, err)
	require.Equal(t, hash, same)
	event.Payload = json.RawMessage(`{"a":9007199254740992,"b":2}`)
	different, err := event.Fingerprint()
	require.NoError(t, err)
	require.NotEqual(t, hash, different)
	require.Equal(t, id, event.ID())
	event.WorkspaceID = "w2"
	require.NotEqual(t, id, event.ID())
	require.NotEqual(t, stableID("ntf", "a", "bc"), stableID("ntf", "ab", "c"))
	require.NotEqual(t, DeliveryID(id, Email, "alice"), DeliveryID(id, Email, "bob"))
	require.NotEqual(t, DeliveryID(id, Slack, "one"), DeliveryID(id, Discord, "one"))
}

func TestEventValidation(t *testing.T) {
	for _, payload := range []string{"", "null", " null ", "[]", "1", `{"a":1} {"b":2}`} {
		t.Run(payload, func(t *testing.T) {
			event := Event{WorkspaceID: "w", Type: "test", SourceEventID: "e", SchemaVersion: 1, OccurredAt: 1, Payload: json.RawMessage(payload)}
			require.Error(t, event.Validate())
		})
	}
	event := Event{WorkspaceID: "w", Type: "test", SourceEventID: "e", SchemaVersion: 1, OccurredAt: 1, Payload: json.RawMessage(`{}`), Resource: Resource{AppID: "app"}}
	require.Error(t, event.Validate())
	event.Resource.ProjectID = "project"
	require.NoError(t, event.Validate())
}

func TestTypedDefinition(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	definition := JSONDefinition[payload]{Name: "test", SchemaVersion: 3, RenderPayload: func(p payload) (Content, error) { return Content{Subject: p.Name}, nil }}
	event := Event{Type: "test", SchemaVersion: 3, Payload: json.RawMessage(`{"name":"API"}`)}
	content, err := definition.Render(event)
	require.NoError(t, err)
	require.Equal(t, "API", content.Subject)
	event.SchemaVersion = 2
	_, err = definition.Render(event)
	require.Error(t, err)
	event.SchemaVersion = 3
	event.Payload = json.RawMessage(`{"name":"API","extra":1}`)
	_, err = definition.Render(event)
	require.Error(t, err)
}

func TestDuplicateObjectKeysAreRejected(t *testing.T) {
	payloads := []string{
		`{"appName":"first","appName":null}`,
		`{"appName":null,"appName":"first"}`,
		`{"nested":{"name":1,"name":2}}`,
		`{"items":[{"name":1,"name":2}]}`,
		`{"name":1,"na\u006de":2}`,
	}
	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			_, err := validEvent(payload).Normalize()
			require.Error(t, err)
		})
	}

	event := validEvent(`{"left":{"name":1},"right":{"name":2}}`)
	normalized, err := event.Normalize()
	require.NoError(t, err)
	require.JSONEq(t, string(event.Payload), string(normalized.Payload))
}

func TestDefinitionUsesCanonicalPayload(t *testing.T) {
	type payload struct {
		Name    string  `json:"name"`
		AppName *string `json:"appName"`
	}
	definition := JSONDefinition[payload]{Name: "test", SchemaVersion: 1, RenderPayload: func(p payload) (Content, error) {
		return Content{Subject: p.Name}, nil
	}}

	first := validEvent(`{"name":"lower","Name":"upper"}`)
	second := validEvent(`{"Name":"upper","name":"lower"}`)
	firstHash, err := first.Fingerprint()
	require.NoError(t, err)
	secondHash, err := second.Fingerprint()
	require.NoError(t, err)
	require.Equal(t, firstHash, secondHash)
	firstContent, err := definition.Render(first)
	require.NoError(t, err)
	secondContent, err := definition.Render(second)
	require.NoError(t, err)
	require.Equal(t, firstContent, secondContent)

	for _, raw := range []string{`{"appName":"first","appName":null}`, `{"appName":null,"appName":"first"}`} {
		_, err := definition.Render(Event{Type: "test", SchemaVersion: 1, Payload: json.RawMessage(raw)})
		require.Error(t, err)
	}
}

func TestNormalizeCanonicalizesWithoutLosingIntegerPrecision(t *testing.T) {
	normalized, err := validEvent(`{"z":1,"a":9007199254740993123456789}`).Normalize()
	require.NoError(t, err)
	require.Equal(t, `{"a":9007199254740993123456789,"z":1}`, string(normalized.Payload))
}

func TestNormalizePayloadLimits(t *testing.T) {
	normalized, err := validEvent(`{"v":"` + strings.Repeat("<", (64*1024-8)/6) + `"}`).Normalize()
	require.NoError(t, err)
	repeated, err := normalized.Normalize()
	require.NoError(t, err)
	require.Equal(t, normalized, repeated)
	_, err = validEvent(`{"v":"` + strings.Repeat("<", (64*1024-8)/6+1) + `"}`).Normalize()
	require.Error(t, err)
	_, err = validEvent(`{"v":` + strings.Repeat("[", 10001) + `0` + strings.Repeat("]", 10001) + `}`).Normalize()
	require.Error(t, err)
}

func validEvent(payload string) Event {
	return Event{
		WorkspaceID:   "w",
		SourceEventID: "e",
		Type:          "test",
		SchemaVersion: 1,
		Payload:       json.RawMessage(payload),
		OccurredAt:    1,
	}
}
