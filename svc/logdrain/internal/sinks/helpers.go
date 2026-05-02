package sinks

import (
	"bytes"
	"encoding/json"
	"time"
)

// SourceLabel is the canonical short name attached to each forwarded record
// so the customer can split runtime logs from access logs in their
// provider with a single facet. Exported so per-provider subpackages
// can stamp it onto their wire format consistently.
func SourceLabel(k RecordKind) string {
	switch k {
	case RecordRequest:
		return "request"
	case RecordRuntime:
		fallthrough
	default:
		return "runtime"
	}
}

// HealthCheckRecord is the synthetic record sent by Sink.HealthCheck to
// verify auth and dataset/endpoint configuration at drain creation. It
// is shaped like a real runtime record so the provider's parser does not
// silently reject a "weird" payload — failures bubble up as auth or
// quota errors, which is what the dashboard test-push needs to surface.
func HealthCheckRecord() Record {
	return Record{
		Kind:          RecordRuntime,
		TimeMs:        time.Now().UnixMilli(),
		SeverityText:  "info",
		WorkspaceID:   "ws_healthcheck",
		ProjectID:     "proj_healthcheck",
		EnvironmentID: "",
		AppID:         "",
		DeploymentID:  "",
		Region:        "",
		Platform:      "",
		K8sPodName:    "",
		Body:          "unkey logdrain test push",
		Attributes: map[string]any{
			"unkey.healthcheck": true,
		},
		// Synthetic record with no source-level cursor; the value is
		// unused because the worker doesn't run health-check sends
		// through the per-drain cursor advance path.
		CursorTimeMs: 0,
		// Health-check pushes have no row to dedup against; an empty
		// LastID surfaces as an empty Idempotency-Key downstream.
		LastID: "",
	}
}

// EncodeJSON marshals each record once via the caller-provided mapper.
// Pre-encoding lets a sink walk the slice in one pass to compute chunk
// boundaries from the actual on-the-wire byte count without re-marshalling
// — which is what `ChunkByBytes` needs.
func EncodeJSON[T any](batch []Record, mapper func(Record) T) ([][]byte, error) {
	out := make([][]byte, len(batch))
	for i, r := range batch {
		raw, err := json.Marshal(mapper(r))
		if err != nil {
			return nil, err
		}
		out[i] = raw
	}

	return out, nil
}

// BuildJSONArray joins pre-encoded JSON values with commas and wraps them
// in `[ ... ]`. All three v1 sinks ship batches as a top-level JSON array
// over HTTP, so the wrapping is centralised here.
func BuildJSONArray(chunk [][]byte) []byte {
	return append(append([]byte{'['}, bytes.Join(chunk, []byte{','})...), ']')
}

// ChunkByBytes walks pre-encoded JSON records and invokes flush once per
// chunk that fits within maxChunkBytes (and optionally maxRecords). It
// accounts for the 2 surrounding brackets and the per-record comma
// separators, matching the way providers measure JSON-array request
// bodies.
//
// Pass maxRecords <= 0 to disable the per-chunk record cap (byte-only ceiling).
func ChunkByBytes(encoded [][]byte, maxChunkBytes, maxRecords int, flush func(chunk [][]byte) error) error {
	chunkStart := 0
	chunkBytes := 2 // surrounding `[]`
	for i, raw := range encoded {
		recordBytes := len(raw)
		if i > chunkStart {
			recordBytes++ // comma separator
		}
		overByteCap := chunkStart < i && chunkBytes+recordBytes > maxChunkBytes
		overRecordCap := maxRecords > 0 && i-chunkStart >= maxRecords
		if overByteCap || overRecordCap {
			if err := flush(encoded[chunkStart:i]); err != nil {
				return err
			}
			chunkStart = i
			chunkBytes = 2 + len(raw)
			continue
		}
		chunkBytes += recordBytes
	}
	if chunkStart < len(encoded) {
		return flush(encoded[chunkStart:])
	}
	return nil
}
