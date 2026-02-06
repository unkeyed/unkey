---
title: auditlog
description: "defines types and constants for the audit logging system"
---

Package auditlog defines types and constants for the audit logging system.

Audit logs provide a secure, immutable record of actions taken within the system, tracking who did what and when. This package contains standard definitions to ensure consistency across the audit logging system.

package auditlog contains types and validation logic for audit logs

## Types

### type AuditLog

```go
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
```

AuditLog represents an pretty struct of an audit log entry that we will write into the db

### type AuditLogActor

```go
type AuditLogActor string
```

AuditLogActor represents the type of entity that performed an action. Actors are categorized to enable filtering and analysis of audit logs based on the source of actions.

```go
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
```

### type AuditLogEvent

```go
type AuditLogEvent string
```

AuditLogEvent represents the possible events in the audit log system

```go
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
```

### type AuditLogResource

```go
type AuditLogResource struct {
	ID          string
	DisplayName string
	Name        string
	// json encoded metadata
	Meta map[string]any
	Type AuditLogResourceType
}
```

AuditLogResource represents a single resource that was affected by the action

### type AuditLogResourceType

```go
type AuditLogResourceType string
```

AuditLogResourceType represents the possible type in the audit log targets

```go
const (
	APIResourceType                AuditLogResourceType = "api"
	AuditLogBucketResourceType     AuditLogResourceType = "auditLogBucket"
	IdentityResourceType           AuditLogResourceType = "identity"
	KeySpaceResourceType           AuditLogResourceType = "keySpace"
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
```

