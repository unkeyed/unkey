package deploy_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// deployFixture is the seeded hierarchy a deployment row hangs off.
type deployFixture struct {
	seeder        *seed.Seeder
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func newDeployFixture(t *testing.T, ctx context.Context) (db.Database, deployFixture) {
	t.Helper()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID

	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "KEBAP",
		Slug:        deploySlug(uid.ProjectPrefix),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "KEBAP",
		Slug:          deploySlug(uid.AppPrefix),
		DefaultBranch: "main",
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})

	// A create refuses an environment with nowhere to schedule, so the fixture
	// carries the one region a deployable app has.
	region := seeder.CreateRegion(ctx, seed.CreateRegionRequest{Name: "kebap-1", Platform: "k8s"})
	require.NoError(t, database.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		RegionID:      region.ID,
		Replicas:      1,
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
	}))

	return database, deployFixture{
		seeder:        seeder,
		workspaceID:   workspaceID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

func deploySlug(prefix uid.Prefix) string {
	return strings.ToLower(strings.ReplaceAll(uid.New(prefix), "_", "-"))
}
