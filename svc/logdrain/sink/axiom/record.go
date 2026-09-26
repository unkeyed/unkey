package axiom

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

type recordMetadata struct {
	Stream string `json:"stream"`
	Time   string `json:"_time"`
}

type auditLogRecord struct {
	recordMetadata
	sink.AuditLogPayload
}

type keyVerificationRecord struct {
	recordMetadata
	sink.KeyVerificationPayload
}

type gatewayRequestRecord struct {
	recordMetadata
	sink.GatewayRequestPayload
}

type runtimeLogRecord struct {
	recordMetadata
	sink.RuntimeLogPayload
}

type ratelimitRecord struct {
	recordMetadata
	sink.RatelimitPayload
}

func marshalRecord(event sink.Event) (json.RawMessage, error) {
	metadata := recordMetadata{
		Stream: event.Stream,
		Time:   time.UnixMilli(event.Time).UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	switch payload := event.Payload.(type) {
	case sink.AuditLogPayload:
		return json.Marshal(auditLogRecord{
			recordMetadata:  metadata,
			AuditLogPayload: payload,
		})
	case sink.KeyVerificationPayload:
		return json.Marshal(keyVerificationRecord{
			recordMetadata:         metadata,
			KeyVerificationPayload: payload,
		})
	case sink.GatewayRequestPayload:
		return json.Marshal(gatewayRequestRecord{
			recordMetadata:        metadata,
			GatewayRequestPayload: payload,
		})
	case sink.RuntimeLogPayload:
		return json.Marshal(runtimeLogRecord{
			recordMetadata:    metadata,
			RuntimeLogPayload: payload,
		})
	case sink.RatelimitPayload:
		return json.Marshal(ratelimitRecord{
			recordMetadata:   metadata,
			RatelimitPayload: payload,
		})
	default:
		return nil, fmt.Errorf("unsupported logdrain payload %T", event.Payload)
	}
}
