package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

type Ratelimits struct{ client *clickhouse.Client }

func NewRatelimits(client *clickhouse.Client) *Ratelimits {
	return &Ratelimits{client: client}
}

func (s *Ratelimits) Read(ctx context.Context, workspaceID string, from Cursor, toExclusive int64, limit int, cfg *logdrainv1.Config) ([]sink.Event, Cursor, error) {
	const query = `SELECT inserted_at, time, event_id, request_id, check_index,
		namespace_id, identifier, passed, override_id, limit, remaining, reset_at, tokens
		FROM ratelimits_raw_v2 WHERE workspace_id = {workspace:String}
		AND (inserted_at > {from_time:Int64}
		  OR (inserted_at = {from_time:Int64} AND event_id > {from_id:String}))
		AND inserted_at < {to:Int64}
		AND (empty({namespaces:Array(String)}) OR namespace_id IN {namespaces:Array(String)})
		AND (empty({passed:Array(Bool)}) OR passed IN {passed:Array(Bool)})
		ORDER BY inserted_at, event_id LIMIT {batch_size:UInt64}`
	type row struct {
		InsertedAt  int64  `ch:"inserted_at"`
		Time        int64  `ch:"time"`
		EventID     string `ch:"event_id"`
		RequestID   string `ch:"request_id"`
		CheckIndex  uint32 `ch:"check_index"`
		NamespaceID string `ch:"namespace_id"`
		Identifier  string `ch:"identifier"`
		Passed      bool   `ch:"passed"`
		OverrideID  string `ch:"override_id"`
		Limit       uint64 `ch:"limit"`
		Remaining   uint64 `ch:"remaining"`
		ResetAt     int64  `ch:"reset_at"`
		Tokens      uint64 `ch:"tokens"`
	}
	filters := cfg.GetRatelimits()
	passed := "[]"
	if len(filters.GetPassed()) > 0 {
		encoded, err := json.Marshal(filters.GetPassed())
		if err != nil {
			return nil, from, fmt.Errorf("encode rate-limit results: %w", err)
		}
		passed = string(encoded)
	}
	rows, err := clickhouse.Select[row](ctx, s.client.Conn(), query, map[string]string{
		"workspace":  workspaceID,
		"from_time":  strconv.FormatInt(from.Time, 10),
		"from_id":    from.EventID,
		"to":         strconv.FormatInt(toExclusive, 10),
		"batch_size": strconv.Itoa(limit),
		"namespaces": clickhouse.StringArrayParam(filters.GetNamespaceIds()),
		"passed":     passed,
	})
	if err != nil {
		return nil, from, fmt.Errorf("read rate limits: %w", err)
	}
	events := make([]sink.Event, 0, len(rows))
	next := from
	for _, row := range rows {
		payload := sink.RatelimitPayload{
			RequestID: row.RequestID, CheckIndex: row.CheckIndex,
			NamespaceID: row.NamespaceID, Identifier: row.Identifier,
			Passed: row.Passed, OverrideID: row.OverrideID, Limit: row.Limit,
			Remaining: row.Remaining, ResetAt: row.ResetAt, Tokens: row.Tokens, Source: "api",
		}
		events = append(events, sink.Event{EventID: row.EventID, Stream: "ratelimits", Time: row.Time, Payload: payload})
		next = Cursor{Time: row.InsertedAt, EventID: row.EventID}
	}
	return events, next, nil
}
