package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// mapAuditLog converts a stored ClickHouse row into the stable, SIEM-friendly
// wire shape: epoch-ms timestamps become RFC3339 UTC, the parallel target
// arrays become a resources list, and the JSON meta blobs are decoded into
// objects. outcome is a constant today (only successful platform mutations are
// logged); see the endpoint's KTD5.
func mapAuditLog(r schema.AuditLogV1) openapi.AuditLog {
	al := openapi.AuditLog{ //nolint:exhaustruct // Actor, Context, and Resources are populated below
		AuditLogId:    r.EventID,
		Version:       eventVersion,
		Time:          time.UnixMilli(r.Time).UTC(),
		InsertedAt:    time.UnixMilli(r.InsertedAt).UTC(),
		Event:         r.Event,
		Description:   r.Description,
		Outcome:       openapi.Success,
		Source:        r.Source,
		CorrelationId: strPtrIfSet(r.CorrelationID),
		Meta:          toMetaPtr(r.Meta),
	}

	al.Actor.Type = r.ActorType
	al.Actor.Id = r.ActorID
	al.Actor.Name = strPtrIfSet(r.ActorName)
	al.Actor.Meta = toMetaPtr(r.ActorMeta)

	al.Context.IpAddress = strPtrIfSet(r.RemoteIP)
	al.Context.UserAgent = strPtrIfSet(r.UserAgent)

	// resources is a required, non-nullable array; start empty so it never
	// marshals to null.
	al.Resources = make([]struct {
		Id   string                  `json:"id"`
		Meta *map[string]interface{} `json:"meta,omitempty"`
		Name *string                 `json:"name,omitempty"`
		Type string                  `json:"type"`
	}, 0, len(r.TargetIDs))
	for i := range r.TargetIDs {
		al.Resources = append(al.Resources, struct {
			Id   string                  `json:"id"`
			Meta *map[string]interface{} `json:"meta,omitempty"`
			Name *string                 `json:"name,omitempty"`
			Type string                  `json:"type"`
		}{
			Id:   r.TargetIDs[i],
			Type: sliceAt(r.TargetTypes, i),
			Name: strPtrIfSet(sliceAt(r.TargetNames, i)),
			Meta: toMetaPtr(rawAt(r.TargetMetas, i)),
		})
	}

	return al
}

// sliceAt returns the element at i, or the zero value if i is out of range.
// The parallel target arrays are meant to be aligned, but a defensive read
// keeps a malformed row from panicking the whole page.
func sliceAt(ss []string, i int) string {
	if i < 0 || i >= len(ss) {
		return ""
	}
	return ss[i]
}

func rawAt(rs []json.RawMessage, i int) json.RawMessage {
	if i < 0 || i >= len(rs) {
		return nil
	}
	return rs[i]
}

func strPtrIfSet(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// toMetaPtr decodes a JSON object blob into a map, returning nil for empty,
// null, or "{}" so those fields are omitted from the response rather than
// emitted as empty objects.
func toMetaPtr(raw json.RawMessage) *map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	switch strings.TrimSpace(string(raw)) {
	case "", "{}", "null":
		return nil
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	if len(m) == 0 {
		return nil
	}
	return &m
}
