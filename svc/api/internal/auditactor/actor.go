// Package auditactor maps an authenticated principal to the actor recorded on
// audit logs. It lives in the api layer because the pkg/auditlog package does
// not know about auth principals, mirroring ctrlclient.Actor which does the
// same mapping for ctrl RPCs.
package auditactor

import (
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
)

// Actor is the actor attribution for an audit log entry, ready to copy onto the
// ActorType/ActorID/ActorName/ActorMeta fields of an auditlog.AuditLog.
type Actor struct {
	// Type classifies the actor for filtering and analysis.
	Type auditlog.AuditLogActor

	// ID is the stable identifier of the actor. For portal end users this is
	// the customer-assigned externalId.
	ID string

	// Name is the human-readable actor name.
	Name string

	// Meta is additional actor metadata. Always a non-nil map so callers can
	// assign it directly to AuditLog.ActorMeta.
	Meta map[string]any
}

// FromPrincipal derives the audit log actor from the authenticated principal.
//
// Portal-session principals are attributed to a PortalEndUserActor so customers
// can see end-user activity in their audit logs. All other principals are
// attributed to their subject type. The ID and Name come from the principal's
// subject, which the resolver already sets to the end user's externalId for
// portal sessions.
//
// Keeping actor construction here ensures every portal-capable handler
// attributes actions consistently instead of reaching into principal.Subject
// and repeating the source switch. It is also the single place to update if the
// portal principal contract later makes the subject ID diverge from the
// externalId.
func FromPrincipal(p *principal.Principal) Actor {
	actor := Actor{
		Type: auditlog.AuditLogActor(p.Subject.Type),
		ID:   p.Subject.ID,
		Name: p.Subject.Name,
		Meta: map[string]any{},
	}

	if _, ok := p.Source.(principal.PortalSessionSource); ok {
		actor.Type = auditlog.PortalEndUserActor
	}

	return actor
}
