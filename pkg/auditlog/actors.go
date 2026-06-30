package auditlog

import (
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
)

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

	// PortalEndUserActor indicates the action was performed by an end user
	// authenticated through a customer portal session, rather than by the
	// workspace owner. The actor metadata carries the portal externalId so
	// customers can see what their end users did.
	PortalEndUserActor AuditLogActor = "portalEndUser"
)

// ActorAttribution describes how an authenticated principal should appear as
// the actor on an audit log entry.
type ActorAttribution struct {
	// Type is the actor category to record on the audit log.
	Type AuditLogActor

	// Meta is the actor metadata to record on the audit log. It is always a
	// non-nil map so callers can assign it directly to AuditLog.ActorMeta.
	Meta map[string]any
}

// ActorFromPrincipal derives the audit log actor attribution from the
// authenticated principal.
//
// Portal-session principals are attributed to a PortalEndUserActor with the
// end user's externalId in the metadata, so customers can see end-user
// activity in their audit logs. All other principals are attributed to their
// subject type with empty metadata.
//
// Keeping this derivation in one place ensures every portal-capable handler
// attributes actions consistently instead of repeating the source switch.
func ActorFromPrincipal(p *authprincipal.Principal) ActorAttribution {
	if src, ok := p.Source.(authprincipal.PortalSessionSource); ok {
		return ActorAttribution{
			Type: PortalEndUserActor,
			Meta: map[string]any{
				"externalId": src.ExternalID,
			},
		}
	}

	return ActorAttribution{
		Type: AuditLogActor(p.Subject.Type),
		Meta: map[string]any{},
	}
}
