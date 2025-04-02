package auditlogs

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/auditlog"
)

type AuditLogService interface {
	Insert(ctx context.Context, tx *sql.Tx, logs []auditlog.AuditLog) error
}
