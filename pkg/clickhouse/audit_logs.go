package clickhouse

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

// InsertAuditLogs writes a batch of audit log rows to audit_logs_raw_v1 and
// returns only after ClickHouse confirms the insert. The outbox worker relies
// on this synchronous confirmation before marking the source MySQL rows as
// exported. A retry can create a duplicate row because this path provides
// at-least-once delivery.
func (c *Client) InsertAuditLogs(ctx context.Context, rows []schema.AuditLogV1) error {
	if len(rows) == 0 {
		return nil
	}
	return flush(c, ctx, rows)
}
