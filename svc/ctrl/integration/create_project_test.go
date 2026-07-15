//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/project"
)

// TestCreateProjectWritesAuditLog verifies that CreateProject inserts the
// project and its project.create audit log atomically: the audit row only
// reaches the outbox if the surrounding transaction committed.
func TestCreateProjectWritesAuditLog(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	const bearer = "test-token"

	auditSvc, err := auditlogs.New(auditlogs.Config{DB: h.DB})
	require.NoError(t, err)

	svc := project.New(project.Config{
		Database:  h.DB,
		Auditlogs: auditSvc,
		Bearer:    bearer,
	})

	workspaceID := h.Resources().UserWorkspace.ID
	slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))

	req := connect.NewRequest(&ctrlv1.CreateProjectRequest{
		WorkspaceId: workspaceID,
		Name:        "Payments",
		Slug:        slug,
		Actor: &ctrlv1.ActorInfo{
			Id:        "user_test",
			Name:      "Test User",
			Type:      ctrlv1.ActorType_ACTOR_TYPE_USER,
			RemoteIp:  "127.0.0.1",
			UserAgent: "test-agent",
		},
	})
	req.Header().Set("Authorization", "Bearer "+bearer)

	res, err := svc.CreateProject(ctx, req)
	require.NoError(t, err)
	projectID := res.Msg.GetId()
	require.True(t, strings.HasPrefix(projectID, "proj_"))

	_, err = h.DB.FindProjectById(ctx, projectID)
	require.NoError(t, err, "project row should be committed")

	rows, err := h.DB.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, err)

	var found bool
	for _, row := range rows {
		var ev auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &ev))
		if ev.Event != string(auditlog.ProjectCreateEvent) {
			continue
		}
		for _, target := range ev.Targets {
			if target.ID == projectID {
				found = true
				require.Equal(t, workspaceID, ev.WorkspaceID)
				require.Equal(t, "user_test", ev.Actor.ID)
				require.Equal(t, string(auditlog.UserActor), ev.Actor.Type)
			}
		}
	}
	require.True(t, found, "expected a project.create audit log for the new project")
}
