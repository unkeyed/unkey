package auditlog

// AuditLogTargetType represents the possible type in the audit log targets
type AuditLogTargetType string

const (
	APITargetType                AuditLogTargetType = "api"
	AuditLogBucketTargetType     AuditLogTargetType = "auditLogBucket"
	IdentityTargetType           AuditLogTargetType = "identity"
	KeySpaceTargetType           AuditLogTargetType = "keySpace"
	KeyTargetType                AuditLogTargetType = "key"
	PermissionTargetType         AuditLogTargetType = "permission"
	RatelimitTargetType          AuditLogTargetType = "ratelimit"
	RatelimitNamespaceTargetType AuditLogTargetType = "ratelimitNamespace"
	RatelimitOverrideTargetType  AuditLogTargetType = "ratelimitOverride"
	RoleTargetType               AuditLogTargetType = "role"
	WorkspaceTargetType          AuditLogTargetType = "workspace"
	PortalTargetType             AuditLogTargetType = "portal"
	PortalSessionTargetType      AuditLogTargetType = "portalSession"
	PortalDomainTargetType       AuditLogTargetType = "portalDomain"
	DeploymentTargetType         AuditLogTargetType = "deployment"
	ProjectTargetType            AuditLogTargetType = "project"
	AppTargetType                AuditLogTargetType = "app"
	EnvironmentTargetType        AuditLogTargetType = "environment"
	DomainTargetType             AuditLogTargetType = "domain"
	CLISessionTargetType         AuditLogTargetType = "cliSession"
)
