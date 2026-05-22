package testutil

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
)

// FindAuditLogsByTargetID returns every audit log envelope queued in
// clickhouse_outbox for the harness workspace whose target list includes
// targetID. Audit logs land in MySQL only as outbox rows now; the drainer
// ships them to ClickHouse out of band.
func (h *Harness) FindAuditLogsByTargetID(ctx context.Context, t *testing.T, targetID string) []auditlog.Event {
	t.Helper()

	rows, err := db.Query.ListClickhouseOutboxByWorkspace(ctx, h.DB.RO(), h.Resources().UserWorkspace.ID)
	require.NoError(t, err)

	var matches []auditlog.Event
	for _, row := range rows {
		if row.Version != auditlog.OutboxVersionV1 {
			continue
		}
		var ev auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &ev), "unmarshal outbox payload pk=%d", row.Pk)
		for _, target := range ev.Targets {
			if target.ID == targetID {
				matches = append(matches, ev)
				break
			}
		}
	}
	return matches
}
