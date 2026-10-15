package testutil

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
)

// FindAuditLogs returns audit events queued in the harness workspace's outbox.
func (h *Harness) FindAuditLogs(ctx context.Context, t *testing.T) []auditlog.Event {
	t.Helper()

	rows, err := db.Query.ListClickhouseOutboxByWorkspace(ctx, h.DB.RO(), h.Resources().UserWorkspace.ID)
	require.NoError(t, err)

	var events []auditlog.Event
	for _, row := range rows {
		if row.Version != auditlog.OutboxVersionV1 {
			continue
		}
		var ev auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &ev), "unmarshal outbox payload pk=%d", row.Pk)
		events = append(events, ev)
	}
	return events
}

// FindAuditLogsByTargetID returns queued audit events in the harness workspace
// whose target list includes targetID.
func (h *Harness) FindAuditLogsByTargetID(ctx context.Context, t *testing.T, targetID string) []auditlog.Event {
	t.Helper()

	var matches []auditlog.Event
	for _, ev := range h.FindAuditLogs(ctx, t) {
		for _, target := range ev.Targets {
			if target.ID == targetID {
				matches = append(matches, ev)
				break
			}
		}
	}
	return matches
}
