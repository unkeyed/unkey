// Package auditlogs provides audit logging services for tracking user actions
// and system events across the Unkey platform. This service ensures compliance,
// security monitoring, and operational visibility by creating immutable records
// of all significant operations.
package auditlogs

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
)

// AuditLogService provides methods for creating and managing audit log entries.
// The service handles batch insertion of audit logs with their associated
// resource targets, ensuring transactional consistency and proper data
// normalization for efficient querying and compliance reporting.
//
// AuditLogService is used across all Unkey services including the API service
// for key operations, admin service for workspace management, and authentication
// service for access control events. Common usage scenarios include:
//   - API key creation, modification, and deletion
//   - Permission and role changes
//   - Workspace configuration updates
//   - Authentication and authorization events
//   - System configuration changes
//
// The service automatically handles transaction management when no transaction
// is provided, ensuring that audit logs are never lost due to partial failures.
// When used within existing transactions, audit logs become part of the
// broader operation's atomicity guarantees.
type AuditLogService interface {
	// Insert creates audit log entries for the provided logs and their associated
	// resources within the given transaction context. Insert handles the complete
	// audit log lifecycle including ID generation, timestamp assignment, metadata
	// serialization, and database persistence.
	//
	// The ctx parameter provides request lifecycle management including timeouts
	// and cancellation. The tx parameter can be nil, in which case Insert will
	// create its own transaction for atomic log insertion. When tx is provided,
	// the audit logs become part of the existing transaction's scope.
	//
	// The logs parameter contains the audit events to be recorded, including
	// actor information, affected resources, and contextual metadata. Each log
	// can affect multiple resources, and the service will create appropriate
	// target records for efficient querying.
	//
	// Returns nil on successful insertion of all logs and their targets. Returns
	// an error if any step fails, including JSON serialization of metadata,
	// database insertion failures, or transaction management issues. When using
	// an internal transaction, failures trigger automatic rollback.
	//
	// Insert is safe for concurrent use and handles batch operations efficiently.
	// The method automatically assigns default bucket values and generates unique
	// identifiers for each audit log entry to prevent conflicts.
	//
	// Empty log slices are handled gracefully and return immediately without
	// database interaction. This allows callers to batch audit logs without
	// checking for empty conditions.
	//
	// Example usage within an existing transaction:
	//
	//	err := auditSvc.Insert(ctx, tx, []auditlog.AuditLog{
	//		{
	//			Event:       auditlog.APICreateEvent,
	//			WorkspaceID: workspaceID,
	//			Display:     "Created API production-payments",
	//			ActorID:     userID,
	//			ActorType:   auditlog.UserActor,
	//			ActorName:   "john.doe@company.com",
	//			RemoteIP:    clientIP,
	//			UserAgent:   userAgent,
	//			Resources:   []auditlog.AuditLogResource{
	//				{
	//					ID:          apiID,
	//					Type:        auditlog.APIResourceType,
	//					DisplayName: "production-payments",
	//				},
	//			},
	//		},
	//	})
	//
	// See [auditlog.AuditLog] for log structure details and [db.DBTX] for
	// transaction interface information.
	Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error
}
