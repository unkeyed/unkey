package app

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestCreateAppDockerSource verifies that apps are docker-image apps by
// default and that a caller-supplied default image is persisted in
// app_docker_sources within the create transaction.
func TestCreateAppDockerSource(t *testing.T) {
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
		Name:        "Docker Source Apps",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("project"), "_", "-")),
	})
	seeder.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     strings.ToLower(strings.ReplaceAll(uid.New("region"), "_", "-")),
		Platform: "test",
	})

	audit, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	svc := New(Config{
		Database:  database,
		Restate:   nil,
		Auditlogs: audit,
		Bearer:    bearer,
	})

	createApp := func(t *testing.T, dockerSource *ctrlv1.DockerSource) (*connect.Response[ctrlv1.CreateAppResponse], error) {
		t.Helper()
		req := connect.NewRequest(&ctrlv1.CreateAppRequest{
			WorkspaceId:  workspaceID,
			ProjectId:    project.ID,
			Name:         "Docker App",
			Slug:         strings.ToLower(strings.ReplaceAll(uid.New("docker-app"), "_", "-")),
			DockerSource: dockerSource,
			Actor: &ctrlv1.ActorInfo{
				Id:        "user_test",
				Name:      "Test User",
				Type:      ctrlv1.ActorType_ACTOR_TYPE_USER,
				RemoteIp:  "127.0.0.1",
				UserAgent: "test-agent",
			},
		})
		req.Header().Set("Authorization", "Bearer "+bearer)
		return svc.CreateApp(ctx, req)
	}

	sourceTypeOf := func(t *testing.T, appID string) string {
		t.Helper()
		var sourceType string
		scanErr := database.RW().QueryRowContext(ctx,
			`SELECT source_type FROM apps WHERE id = ?`, appID).Scan(&sourceType)
		require.NoError(t, scanErr)
		return sourceType
	}

	t.Run("saves default image", func(t *testing.T) {
		resp, createErr := createApp(t, &ctrlv1.DockerSource{Image: "ghcr.io/acme/mcp:1.2.3"})
		require.NoError(t, createErr)

		require.Equal(t, "docker_image", sourceTypeOf(t, resp.Msg.GetId()))

		dockerSource, findErr := database.FindAppDockerSourceByAppId(ctx, resp.Msg.GetId())
		require.NoError(t, findErr)
		require.Equal(t, "ghcr.io/acme/mcp:1.2.3", dockerSource.Image)
		require.Equal(t, workspaceID, dockerSource.WorkspaceID)
	})

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		resp, createErr := createApp(t, &ctrlv1.DockerSource{Image: "  ghcr.io/acme/mcp:latest  "})
		require.NoError(t, createErr)

		dockerSource, findErr := database.FindAppDockerSourceByAppId(ctx, resp.Msg.GetId())
		require.NoError(t, findErr)
		require.Equal(t, "ghcr.io/acme/mcp:latest", dockerSource.Image)
	})

	t.Run("rejects empty image", func(t *testing.T) {
		_, createErr := createApp(t, &ctrlv1.DockerSource{Image: ""})
		require.Error(t, createErr)
		var connectErr *connect.Error
		require.ErrorAs(t, createErr, &connectErr)
		require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	})

	t.Run("rejects whitespace-only image", func(t *testing.T) {
		_, createErr := createApp(t, &ctrlv1.DockerSource{Image: "   "})
		require.Error(t, createErr)
		var connectErr *connect.Error
		require.ErrorAs(t, createErr, &connectErr)
		require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	})

	t.Run("omitted docker source creates unconfigured docker app", func(t *testing.T) {
		resp, createErr := createApp(t, nil)
		require.NoError(t, createErr)

		require.Equal(t, "docker_image", sourceTypeOf(t, resp.Msg.GetId()))

		_, findErr := database.FindAppDockerSourceByAppId(ctx, resp.Msg.GetId())
		require.Error(t, findErr)
		require.True(t, db.IsNotFound(findErr), "expected no docker source row, got: %v", findErr)
	})
}
