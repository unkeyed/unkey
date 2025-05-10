// Package auditlogs provides functionality for recording and retrieving audit logs
// within the Unkey system. Audit logs track actions performed on resources, recording
// who did what and when, which is essential for security compliance and debugging.
//
// The service supports recording multiple audit log entries in a single transaction
// to maintain atomicity, and can integrate with existing transactions from other
// services.
package auditlogs

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/auditlog"
)

// AuditLogService defines the interface for audit log operations.
// It provides methods to record audit events that capture actions
// taken on resources within the system.
type AuditLogService interface {
	// Insert records a batch of audit logs, optionally as part of an existing transaction.
	// If tx is nil, the method will create its own transaction.
	//
	// The logs parameter contains the audit events to be recorded, each including:
	// - The actor who performed the action (user, system, API key)
	// - The resources affected by the action
	// - Metadata about the action and affected resources
	//
	// This method ensures all logs are inserted atomically, either all succeed or all fail.
	// If a transaction is provided, the logs will be part of that transaction, allowing
	// audit logs to be committed alongside other database changes.
	//
	// Returns an error if the insertion fails for any reason.
	Insert(ctx context.Context, tx *sql.Tx, logs []auditlog.AuditLog) error
}
