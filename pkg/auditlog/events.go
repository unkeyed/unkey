// package auditlog contains types and validation logic for audit logs
package auditlog

// AuditLogEvent represents the possible events in the audit log system
type AuditLogEvent string

const (
	// Workspace events
	WorkspaceCreateEvent AuditLogEvent = "workspace.create"
	WorkspaceUpdateEvent AuditLogEvent = "workspace.update"
	WorkspaceDeleteEvent AuditLogEvent = "workspace.delete"
	WorkspaceOptInEvent  AuditLogEvent = "workspace.opt_in"

	// API events
	APICreateEvent AuditLogEvent = "api.create"
	APIUpdateEvent AuditLogEvent = "api.update"
	APIDeleteEvent AuditLogEvent = "api.delete"

	// Key events
	KeyCreateEvent AuditLogEvent = "key.create"
	KeyRerollEvent AuditLogEvent = "key.reroll"
	KeyUpdateEvent AuditLogEvent = "key.update"
	KeyDeleteEvent AuditLogEvent = "key.delete"

	// Ratelimit namespace events
	RatelimitNamespaceCreateEvent AuditLogEvent = "ratelimitNamespace.create"
	RatelimitNamespaceUpdateEvent AuditLogEvent = "ratelimitNamespace.update"
	RatelimitNamespaceDeleteEvent AuditLogEvent = "ratelimitNamespace.delete"

	// Vercel integration events
	VercelIntegrationCreateEvent AuditLogEvent = "vercelIntegration.create"
	VercelIntegrationUpdateEvent AuditLogEvent = "vercelIntegration.update"
	VercelIntegrationDeleteEvent AuditLogEvent = "vercelIntegration.delete"

	// Vercel binding events
	VercelBindingCreateEvent AuditLogEvent = "vercelBinding.create"
	VercelBindingUpdateEvent AuditLogEvent = "vercelBinding.update"
	VercelBindingDeleteEvent AuditLogEvent = "vercelBinding.delete"

	// Role events
	RoleCreateEvent AuditLogEvent = "role.create"
	RoleUpdateEvent AuditLogEvent = "role.update"
	RoleDeleteEvent AuditLogEvent = "role.delete"

	// Permission events
	PermissionCreateEvent AuditLogEvent = "permission.create"
	PermissionUpdateEvent AuditLogEvent = "permission.update"
	PermissionDeleteEvent AuditLogEvent = "permission.delete"

	// Authorization events
	AuthConnectRolePermissionEvent    AuditLogEvent = "authorization.connect_role_and_permission"
	AuthDisconnectRolePermissionEvent AuditLogEvent = "authorization.disconnect_role_and_permissions"
	AuthConnectRoleKeyEvent           AuditLogEvent = "authorization.connect_role_and_key"
	AuthDisconnectRoleKeyEvent        AuditLogEvent = "authorization.disconnect_role_and_key"
	AuthConnectPermissionKeyEvent     AuditLogEvent = "authorization.connect_permission_and_key"
	AuthDisconnectPermissionKeyEvent  AuditLogEvent = "authorization.disconnect_permission_and_key"

	// Identity events
	IdentityCreateEvent AuditLogEvent = "identity.create"
	IdentityUpdateEvent AuditLogEvent = "identity.update"
	IdentityDeleteEvent AuditLogEvent = "identity.delete"

	// Ratelimit events
	RatelimitCreateEvent         AuditLogEvent = "ratelimit.create"
	RatelimitUpdateEvent         AuditLogEvent = "ratelimit.update"
	RatelimitDeleteEvent         AuditLogEvent = "ratelimit.delete"
	RatelimitSetOverrideEvent    AuditLogEvent = "ratelimit.set_override"
	RatelimitReadOverrideEvent   AuditLogEvent = "ratelimit.read_override"
	RatelimitDeleteOverrideEvent AuditLogEvent = "ratelimit.delete_override"

	// Audit log bucket events
	AuditLogBucketCreateEvent AuditLogEvent = "auditLogBucket.create"
)
