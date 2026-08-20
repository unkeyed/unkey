package deploy

import (
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestLifecycleAuditEvent(t *testing.T) {
	actor := &ctrlv1.ActorInfo{
		Id:   "key_123",
		Name: "Root Key",
		Type: ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY,
	}
	deployment := db.Deployment{
		ID:            "d_123",
		WorkspaceID:   "ws_123",
		ProjectID:     "p_123",
		AppID:         "app_123",
		EnvironmentID: "env_123",
	}

	require.Equal(t, audit.Event{
		Actor:         actor,
		CorrelationID: "corr_123",
		WorkspaceID:   deployment.WorkspaceID,
		Event:         auditlog.DeploymentStopEvent,
		Display:       "Stopped deployment d_123",
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
	}, lifecycleAuditEvent(
		actor,
		"corr_123",
		deployment,
		auditlog.DeploymentStopEvent,
		"Stopped deployment d_123",
	))
}
