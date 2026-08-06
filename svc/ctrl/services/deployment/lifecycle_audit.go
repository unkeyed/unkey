package deployment

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
)

// recordLifecycleAudit writes the audit entry for a stop, wake, promote, or
// rollback. Call it after the workflow request is accepted.
func (s *Service) recordLifecycleAudit(
	ctx context.Context,
	event auditlog.AuditLogEvent,
	display string,
	workspaceID string,
	deploymentID string,
	meta map[string]any,
	a *ctrlv1.ActorInfo,
) error {
	return s.auditlogs.Insert(ctx, nil, []auditlog.AuditLog{
		{
			Event:         event,
			WorkspaceID:   workspaceID,
			Display:       display,
			ActorID:       a.GetId(),
			ActorType:     actor.AuditType(a.GetType()),
			ActorName:     a.GetName(),
			ActorMeta:     actor.Meta(a.GetMeta()),
			RemoteIP:      a.GetRemoteIp(),
			UserAgent:     a.GetUserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				{
					Type:        auditlog.DeploymentResourceType,
					ID:          deploymentID,
					Name:        "",
					DisplayName: deploymentID,
					Meta:        meta,
				},
			},
		},
	})
}

// lifecycleAuditMeta is the resource metadata shared by every lifecycle entry.
func lifecycleAuditMeta(projectID, appID, environmentID string) map[string]any {
	return map[string]any{
		"projectId":     projectID,
		"appId":         appID,
		"environmentId": environmentID,
	}
}

// auditFailure labels an audit insert error with the action it belongs to.
func auditFailure(action string, err error) error {
	return fmt.Errorf("failed to audit %s: %w", action, err)
}
