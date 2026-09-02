package deploy_test

import (
	"context"
	"strings"
	"testing"

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
