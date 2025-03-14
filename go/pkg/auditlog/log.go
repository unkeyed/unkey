package auditlog

// AuditLog represents an pretty struct of an audit log entry that we will write into the db
type AuditLog struct {
	Event AuditLogEvent

	WorkspaceID string
	Display     string

	Bucket string

	Actor AuditLogActorData

	// There can be multiple resources affected by the action
	Resources []AuditLogResource

	RemoteIP  string
	UserAgent string
}

// AuditLogActorData represents the actor data of who performed the action
type AuditLogActorData struct {
	ID   string
	Type AuditLogActor
	Name string
	Meta []byte
}

// AuditLogResource represents a single resource that was affected by the action
type AuditLogResource struct {
	ID          string
	Name        string
	DisplayName string
	Meta        []byte
	Type        AuditLogResourceType
}
