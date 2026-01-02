package auditlog

// AuditLog represents an pretty struct of an audit log entry that we will write into the db
type AuditLog struct {
	Event       AuditLogEvent
	WorkspaceID string
	Display     string

	ActorID   string
	ActorType AuditLogActor
	ActorName string

	// json encoded metadata
	ActorMeta map[string]any

	// There can be multiple resources affected by the action
	Resources []AuditLogResource

	RemoteIP  string
	UserAgent string
}

// AuditLogResource represents a single resource that was affected by the action
type AuditLogResource struct {
	ID          string
	DisplayName string
	Name        string
	// json encoded metadata
	Meta map[string]any
	Type AuditLogResourceType
}
