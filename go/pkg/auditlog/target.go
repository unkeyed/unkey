package auditlog

// AuditLogResourceType represents the possible type in the audit log targets
type AuditLogResourceType string

const (
	APIResourceType                AuditLogResourceType = "api"
	AuditLogBucketResourceType     AuditLogResourceType = "auditLogBucket"
	IdentityResourceType           AuditLogResourceType = "identity"
	KeyAuthResourceType            AuditLogResourceType = "keyAuth"
	KeyResourceType                AuditLogResourceType = "key"
	PermissionResourceType         AuditLogResourceType = "permission"
	RatelimitResourceType          AuditLogResourceType = "ratelimit"
	RatelimitNamespaceResourceType AuditLogResourceType = "ratelimitNamespace"
	RatelimitOverrideResourceType  AuditLogResourceType = "ratelimitOverride"
	RoleResourceType               AuditLogResourceType = "role"
	VercelBindingResourceType      AuditLogResourceType = "vercelBinding"
	VercelIntegrationResourceType  AuditLogResourceType = "vercelIntegration"
	WorkspaceResourceType          AuditLogResourceType = "workspace"
)
