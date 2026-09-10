package source

import (
	"context"
	"fmt"
	"strconv"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// KeyVerifications reads verification outcomes in ClickHouse insertion order.
type KeyVerifications struct{ client *clickhouse.Client }

func NewKeyVerifications(client *clickhouse.Client) *KeyVerifications {
	return &KeyVerifications{client: client}
}

// Read applies outcome filters before pagination. Empty selects all outcomes.
func (s *KeyVerifications) Read(ctx context.Context, workspaceID string, from Cursor, toExclusive int64, limit int, outcomes []string) ([]sink.Event, Cursor, error) {
	const query = `SELECT inserted_at, time, request_id, key_space_id,
		identity_id, external_id, key_id, region, source, app_id, outcome, tags, spent_credits
		FROM key_verifications_raw_v2
		WHERE workspace_id = {workspace:String}
		AND (inserted_at > {from_time:Int64}
		  OR (inserted_at = {from_time:Int64} AND request_id > {from_id:String}))
		AND inserted_at < {to:Int64}
		AND (empty({outcomes:Array(String)}) OR outcome IN {outcomes:Array(String)})
		ORDER BY inserted_at, request_id LIMIT {batch_size:UInt64}`
	type row struct {
		InsertedAt int64 `ch:"inserted_at"`
		schema.KeyVerification
	}
	rows, err := clickhouse.Select[row](ctx, s.client.Conn(), query, map[string]string{
		"workspace":  workspaceID,
		"from_time":  strconv.FormatInt(from.Time, 10),
		"from_id":    from.EventID,
		"to":         strconv.FormatInt(toExclusive, 10),
		"batch_size": strconv.Itoa(limit),
		"outcomes":   clickhouse.StringArrayParam(outcomes),
	})
	if err != nil {
		return nil, from, fmt.Errorf("read key verifications: %w", err)
	}
	events := make([]sink.Event, 0, len(rows))
	next := from
	for _, row := range rows {
		origin := sink.KeyVerificationSource{Type: row.Source, AppID: ""}
		if row.Source == schema.SourceGateway {
			origin.AppID = row.AppID
		}
		payload := sink.KeyVerificationPayload{
			RequestID:    row.RequestID,
			KeySpaceID:   row.KeySpaceID,
			Identity:     sink.KeyVerificationIdentity{ID: row.IdentityID, ExternalID: row.ExternalID},
			KeyID:        row.KeyID,
			Region:       row.Region,
			Source:       origin,
			Outcome:      row.Outcome,
			Tags:         row.Tags,
			SpentCredits: row.SpentCredits,
		}
		events = append(events, sink.Event{EventID: row.RequestID, Stream: "key_verifications", Time: row.Time, Payload: payload})
		next = Cursor{Time: row.InsertedAt, EventID: row.RequestID}
	}
	return events, next, nil
}
