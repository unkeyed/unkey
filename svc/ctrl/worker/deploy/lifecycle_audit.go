package deploy

import (
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// insertLifecycleAudit records a stop, wake, promote, or rollback. Every caller
// of those handlers sends an actor, so a nil actor means a caller forgot one.
// The entry is skipped rather than failing a handler whose state change has
// already applied.
func (w *Workflow) insertLifecycleAudit(
	ctx restate.ObjectContext,
	actor *ctrlv1.ActorInfo,
	correlationID string,
	deployment db.Deployment,
	event auditlog.AuditLogEvent,
	display string,
) error {
	if actor == nil {
		return nil
	}

	return audit.Insert(ctx, w.auditlogs, lifecycleAuditEvent(actor, correlationID, deployment, event, display))
}

func lifecycleAuditEvent(
	actor *ctrlv1.ActorInfo,
	correlationID string,
	deployment db.Deployment,
	event auditlog.AuditLogEvent,
	display string,
) audit.Event {
	return audit.Event{
		Actor:         actor,
		CorrelationID: correlationID,
		WorkspaceID:   deployment.WorkspaceID,
		Event:         event,
		Display:       display,
		Resource: auditlog.AuditLogResource{
			Type:        auditlog.DeploymentResourceType,
			ID:          deployment.ID,
			Name:        "",
			DisplayName: deployment.ID,
			Meta: map[string]any{
				"projectId":     deployment.ProjectID,
				"appId":         deployment.AppID,
				"environmentId": deployment.EnvironmentID,
			},
		},
	}
}
