// Package auditlogs provides the control plane audit log outbox writer.
package auditlogs

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// AuditLogService records control plane audit log entries.
type AuditLogService interface {
	// Insert writes audit log entries to the ClickHouse outbox. When tx is nil,
	// Insert opens its own transaction; otherwise the outbox rows commit with
	// the caller's transaction.
	Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error
}
