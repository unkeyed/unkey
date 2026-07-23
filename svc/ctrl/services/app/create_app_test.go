package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

var errInjectedAuditInsert = errors.New("injected audit insert failure")

// TestCreateAppRollsBackWhenAuditInsertFails verifies the production
// guarantee that app creation is all-or-nothing: the app, default environments,
// environment settings, regional settings, and audit outbox row commit together.
func TestCreateAppRollsBackWhenAuditInsertFails(t *testing.T) {
	ctx := context.Background()
	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	const bearer = "test-token"

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)

	workspaceID := seeder.Resources.UserWorkspace.ID
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "Atomic CreateApp",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("project"), "_", "-")),
	})
	seeder.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     strings.ToLower(strings.ReplaceAll(uid.New("region"), "_", "-")),
		Platform: "test",
	})

	slug := strings.ToLower(strings.ReplaceAll(uid.New("atomic-app"), "_", "-"))
	svc := New(Config{
		Database: database,
		Auditlogs: failingAuditLogService{
			t:           t,
			workspaceID: workspaceID,
			projectID:   project.ID,
			appSlug:     slug,
		},
		Bearer: bearer,
	})

	req := connect.NewRequest(&ctrlv1.CreateAppRequest{
		WorkspaceId:  workspaceID,
		ProjectId:    project.ID,
		Name:         "Atomic App",
		Slug:         slug,
		DockerSource: &ctrlv1.DockerSource{Image: "ghcr.io/acme/atomic:1.0.0"},
		Actor: &ctrlv1.ActorInfo{
			Id:        "user_test",
			Name:      "Test User",
			Type:      ctrlv1.ActorType_ACTOR_TYPE_USER,
			RemoteIp:  "127.0.0.1",
			UserAgent: "test-agent",
		},
	})
	req.Header().Set("Authorization", "Bearer "+bearer)

	_, err = svc.CreateApp(ctx, req)
	require.Error(t, err)
	require.ErrorIs(t, err, errInjectedAuditInsert)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())

	require.Equal(t, 0, countRows(t, ctx, database.RW(), `
		SELECT COUNT(*)
		FROM apps
		WHERE workspace_id = ? AND project_id = ? AND slug = ?
	`, workspaceID, project.ID, slug))
	require.Equal(t, 0, countRows(t, ctx, database.RW(), `
		SELECT COUNT(*)
		FROM environments
		WHERE workspace_id = ? AND project_id = ?
	`, workspaceID, project.ID))
	require.Equal(t, 0, countRows(t, ctx, database.RW(), `
		SELECT COUNT(*)
		FROM app_build_settings
		WHERE workspace_id = ?
	`, workspaceID))
	require.Equal(t, 0, countRows(t, ctx, database.RW(), `
		SELECT COUNT(*)
		FROM app_runtime_settings
		WHERE workspace_id = ?
	`, workspaceID))
	require.Equal(t, 0, countRows(t, ctx, database.RW(), `
		SELECT COUNT(*)
		FROM app_regional_settings
		WHERE workspace_id = ?
	`, workspaceID))
	require.Equal(t, 0, countRows(t, ctx, database.RW(), `
		SELECT COUNT(*)
		FROM app_docker_sources
		WHERE workspace_id = ?
	`, workspaceID))

	outboxRows, err := database.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, err)
	require.Empty(t, outboxRows)
}

func TestPickDefaultRegion(t *testing.T) {
	region := func(id, name string, canSchedule bool) db.ListRegionsRow {
		return db.ListRegionsRow{ID: id, Name: name, Platform: "aws", CanSchedule: canSchedule}
	}

	t.Run("no regions", func(t *testing.T) {
		id, ok := pickDefaultRegion(nil)
		require.False(t, ok)
		require.Empty(t, id)
	})

	t.Run("no schedulable regions", func(t *testing.T) {
		id, ok := pickDefaultRegion([]db.ListRegionsRow{
			region("rgn_1", "us-east-1", false),
			region("rgn_2", "eu-west-1", false),
		})
		require.False(t, ok)
		require.Empty(t, id)
	})

	t.Run("picks lexically first schedulable region", func(t *testing.T) {
		id, ok := pickDefaultRegion([]db.ListRegionsRow{
			region("rgn_use", "us-east-1", true),
			region("rgn_euw", "eu-west-1", true),
			region("rgn_usw", "us-west-2", true),
		})
		require.True(t, ok)
		require.Equal(t, "rgn_euw", id, "eu-west-1 sorts first among schedulable regions")
	})

	t.Run("skips unschedulable region that would sort first", func(t *testing.T) {
		id, ok := pickDefaultRegion([]db.ListRegionsRow{
			region("rgn_aps", "ap-south-1", false),
			region("rgn_euw", "eu-west-1", true),
			region("rgn_usw", "us-west-2", true),
		})
		require.True(t, ok)
		require.Equal(t, "rgn_euw", id, "ap-south-1 sorts first but is not schedulable")
	})

	t.Run("picks any schedulable region regardless of name", func(t *testing.T) {
		id, ok := pickDefaultRegion([]db.ListRegionsRow{
			region("rgn_euw", "eu-west-1", true),
			region("rgn_aps", "ap-south-1", true),
		})
		require.True(t, ok)
		require.Equal(t, "rgn_aps", id, "ap-south-1 sorts before eu-west-1")
	})

	t.Run("single schedulable region wins", func(t *testing.T) {
		id, ok := pickDefaultRegion([]db.ListRegionsRow{
			region("rgn_ap", "ap-south-1", false),
			region("rgn_use", "us-east-1", true),
		})
		require.True(t, ok)
		require.Equal(t, "rgn_use", id, "ap-south-1 sorts first but is not schedulable")
	})
}

type failingAuditLogService struct {
	t           *testing.T
	workspaceID string
	projectID   string
	appSlug     string
}

func (s failingAuditLogService) Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
	s.t.Helper()

	require.NotNil(s.t, tx)
	require.Len(s.t, logs, 1)
	require.Equal(s.t, s.workspaceID, logs[0].WorkspaceID)
	require.Equal(s.t, auditlog.AppCreateEvent, logs[0].Event)
	require.Len(s.t, logs[0].Resources, 1)
	require.Equal(s.t, auditlog.AppResourceType, logs[0].Resources[0].Type)
	require.Equal(s.t, s.appSlug, logs[0].Resources[0].Meta["slug"])
	require.Equal(s.t, s.projectID, logs[0].Resources[0].Meta["projectId"])

	appID := logs[0].Resources[0].ID
	require.Equal(s.t, 1, countRows(s.t, ctx, tx, `
		SELECT COUNT(*)
		FROM apps
		WHERE id = ? AND workspace_id = ? AND project_id = ? AND slug = ?
	`, appID, s.workspaceID, s.projectID, s.appSlug))
	require.Equal(s.t, 2, countRows(s.t, ctx, tx, `
		SELECT COUNT(*)
		FROM environments
		WHERE app_id = ? AND workspace_id = ? AND project_id = ?
	`, appID, s.workspaceID, s.projectID))
	require.Equal(s.t, 2, countRows(s.t, ctx, tx, `
		SELECT COUNT(*)
		FROM app_build_settings
		WHERE app_id = ? AND workspace_id = ?
	`, appID, s.workspaceID))
	require.Equal(s.t, 2, countRows(s.t, ctx, tx, `
		SELECT COUNT(*)
		FROM app_runtime_settings
		WHERE app_id = ? AND workspace_id = ?
	`, appID, s.workspaceID))
	require.Equal(s.t, 2, countRows(s.t, ctx, tx, `
		SELECT COUNT(*)
		FROM app_regional_settings
		WHERE app_id = ? AND workspace_id = ?
	`, appID, s.workspaceID))
	require.Equal(s.t, 1, countRows(s.t, ctx, tx, `
		SELECT COUNT(*)
		FROM app_docker_sources
		WHERE app_id = ? AND workspace_id = ?
	`, appID, s.workspaceID))

	return errInjectedAuditInsert
}

func countRows(t *testing.T, ctx context.Context, tx db.DBTX, query string, args ...any) int {
	t.Helper()

	var count int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err)
	return count
}
