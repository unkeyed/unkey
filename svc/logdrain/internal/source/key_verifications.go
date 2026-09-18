package source

import (
	"context"
	"fmt"
	"strconv"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// KeyVerifications reads verification outcomes in ClickHouse insertion order.
type KeyVerifications struct{ client *clickhouse.Client }

func NewKeyVerifications(client *clickhouse.Client) *KeyVerifications {
	return &KeyVerifications{client: client}
}

// Read applies outcome and keyspace filters before pagination. Empty filters select all values.
func (s *KeyVerifications) Read(ctx context.Context, workspaceID string, from Cursor, toExclusive int64, limit int, config *logdrainv1.Config) ([]sink.Event, Cursor, error) {
	const query = `SELECT inserted_at, time, request_id, key_space_id,
		identity_id, external_id, key_id, region, source, app_id, outcome, tags, spent_credits
		FROM key_verifications_raw_v2
		WHERE workspace_id = {workspace:String}
		AND (inserted_at > {from_time:Int64}
		  OR (inserted_at = {from_time:Int64} AND request_id > {from_id:String}))
		AND inserted_at < {to:Int64}
		AND (empty({outcomes:Array(String)}) OR outcome IN {outcomes:Array(String)})
		AND (empty({key_space_ids:Array(String)}) OR key_space_id IN {key_space_ids:Array(String)})
		ORDER BY inserted_at, request_id LIMIT {batch_size:UInt64}`
	type row struct {
		InsertedAt int64 `ch:"inserted_at"`
		schema.KeyVerification
	}
	rows, err := clickhouse.Select[row](ctx, s.client.Conn(), query, map[string]string{
		"workspace":     workspaceID,
		"from_time":     strconv.FormatInt(from.Time, 10),
		"from_id":       from.EventID,
		"to":            strconv.FormatInt(toExclusive, 10),
		"batch_size":    strconv.Itoa(limit),
		"outcomes":      clickhouse.StringArrayParam(config.GetKeyVerifications().GetOutcomes()),
		"key_space_ids": clickhouse.StringArrayParam(config.GetKeyVerifications().GetKeySpaceIds()),
	})
	if err != nil {
		return nil, from, fmt.Errorf("read key verifications: %w", err)
	}
	events := array.Map(rows, func(row row) sink.Event {
		origin := sink.KeyVerificationSource{Type: row.Source, AppID: ""}
		if row.Source == schema.SourceGateway {
			origin.AppID = row.AppID
		}
		var identity *sink.KeyVerificationIdentity
		if row.IdentityID != "" || row.ExternalID != "" {
			identity = &sink.KeyVerificationIdentity{ID: row.IdentityID, ExternalID: row.ExternalID}
		}
		payload := sink.KeyVerificationPayload{
			RequestID:    row.RequestID,
			KeySpaceID:   row.KeySpaceID,
			Identity:     identity,
			KeyID:        row.KeyID,
			Region:       row.Region,
			Source:       origin,
			Outcome:      row.Outcome,
			Tags:         row.Tags,
			SpentCredits: row.SpentCredits,
		}
		return sink.Event{EventID: row.RequestID, Stream: "key_verifications", Time: row.Time, Payload: payload}
	})
	next := from
	if len(rows) > 0 {
		last := rows[len(rows)-1]
		next = Cursor{Time: last.InsertedAt, EventID: last.RequestID}
	}
	return events, next, nil
}
