package auditlog

type AuditLog struct {
	Event AuditLogEvent

	WorkspaceID string
	Display     string

	Bucket string

	Actor AuditLogActorData

	Resources []AuditLogResource

	RemoteIP  string
	UserAgent string
}

type AuditLogActorData struct {
	ID   string
	Type AuditLogActor
	Name string
	Meta []byte
}

type AuditLogResource struct {
	ID          string
	Name        string
	DisplayName string
	Meta        []byte
	Type        AuditLogResourceType
}
