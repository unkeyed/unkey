package auditlog

// AuditLogActor represents the type of entity that performed an action.
// Actors are categorized to enable filtering and analysis of audit logs
// based on the source of actions.
type AuditLogActor string

const (
	// RootKeyActor indicates the action was performed using a root API key.
	// Root keys can manage workspace resources.
	RootKeyActor AuditLogActor = "rootkey"

	// UserActor indicates the action was performed by a human user
	// directly interacting with the system, typically through the UI.
	UserActor AuditLogActor = "user"

	// SystemActor indicates the action was performed automatically by
	// the system itself, without direct human intervention.
	// This might include scheduled tasks, automatic cleanups, or
	// system maintenance operations.
	SystemActor AuditLogActor = "system"
)
