// Package actor maps the shared ctrl.v1.ActorInfo wire type onto audit log
// fields, so every ctrl RPC that writes audit logs converts actors the same way.
package actor

import (
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
)

// AuditType maps a wire ActorType to its audit log actor type. Unknown and
// unspecified types fall back to the system actor.
func AuditType(t ctrlv1.ActorType) auditlog.AuditLogActor {
	switch t {
	case ctrlv1.ActorType_ACTOR_TYPE_USER:
		return auditlog.UserActor
	case ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY:
		return auditlog.RootKeyActor
	case ctrlv1.ActorType_ACTOR_TYPE_SYSTEM, ctrlv1.ActorType_ACTOR_TYPE_UNSPECIFIED:
		return auditlog.SystemActor
	default:
		return auditlog.SystemActor
	}
}

// Meta converts wire actor metadata to the audit log's untyped map.
func Meta(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
