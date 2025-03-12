package auditlog

// AuditLogResourceType represents the possible type in the audit log targets
type AuditLogResourceType string

const (
	IdentityResourceType  AuditLogResourceType = "identity"
	RatelimitResourceType AuditLogResourceType = "ratelimit"
)
