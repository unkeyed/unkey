package httpdrain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

func TestMarshalRecordNestedFields(t *testing.T) {
	actorMetadata, err := json.Marshal(map[string]int{"actor": 1})
	require.NoError(t, err)
	targetMetadata, err := json.Marshal(map[string]int{"target": 2})
	require.NoError(t, err)
	eventMetadata, err := json.Marshal(map[string]int{"event": 3})
	require.NoError(t, err)
	for _, tt := range []struct {
		name    string
		payload sink.Payload
		want    string
	}{
		{
			name: "audit",
			payload: sink.AuditLogPayload{
				ID:     "event",
				Action: "updated",
				Actor: sink.AuditLogActor{
					ID:       "actor",
					Type:     "user",
					Name:     "Alice",
					Metadata: actorMetadata,
				},
				Targets: []sink.AuditLogTarget{{
					ID:       "target",
					Type:     "key",
					Name:     "Production",
					Metadata: targetMetadata,
				}},
				Context: sink.AuditLogContext{
					Location:  "region",
					UserAgent: "agent",
				},
				Metadata:      eventMetadata,
				Description:   "description",
				CorrelationID: "correlation",
			},
			want: `{"id":"event","action":"updated","actor":{"id":"actor","type":"user","name":"Alice","metadata":{"actor":1}},"targets":[{"id":"target","type":"key","name":"Production","metadata":{"target":2}}],"context":{"location":"region","user_agent":"agent"},"metadata":{"event":3},"description":"description","correlation_id":"correlation"}`,
		},
		{
			name: "verification",
			payload: sink.KeyVerificationPayload{
				RequestID:  "request",
				KeySpaceID: "space",
				Identity: &sink.KeyVerificationIdentity{
					ID:         "identity",
					ExternalID: "external",
				},
				KeyID:  "key",
				Region: "region",
				Source: sink.KeyVerificationSource{
					Type:  "gateway",
					AppID: "app",
				},
				Outcome:      "VALID",
				Tags:         []string{"paid", "production"},
				SpentCredits: 7,
			},
			want: `{"request_id":"request","key_space_id":"space","identity":{"id":"identity","externalId":"external"},"key_id":"key","region":"region","source":{"type":"gateway","appId":"app"},"outcome":"VALID","tags":["paid","production"],"spent_credits":7}`,
		},
		{
			name:    "optional verification fields",
			payload: sink.KeyVerificationPayload{},
			want:    `{"request_id":"","key_space_id":"","key_id":"","region":"","source":{"type":""},"outcome":"","tags":null,"spent_credits":0}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := marshalRecord(sink.Event{
				Stream:  tt.name,
				Time:    123,
				Payload: tt.payload,
			})
			require.NoError(t, err)
			var record map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &record))
			require.JSONEq(t, `"1970-01-01T00:00:00.123Z"`, string(record["time"]))
			require.NotContains(t, record, "_time")
			delete(record, "time")
			delete(record, "stream")
			payload, err := json.Marshal(record)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(payload))
		})
	}
}

func TestMarshalRecordPreservesEmptyTargets(t *testing.T) {
	for _, targets := range [][]sink.AuditLogTarget{nil, {}} {
		encoded, err := marshalRecord(sink.Event{
			Payload: sink.AuditLogPayload{
				Targets: targets,
			},
		})
		require.NoError(t, err)
		var record map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(encoded, &record))
		if targets == nil {
			require.Equal(t, "null", string(record["targets"]))
		} else {
			require.Equal(t, "[]", string(record["targets"]))
		}
	}
}
