package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeletionFindsDescendantsWithMissingParents(t *testing.T) {
	mysql := containers.MySQL(t)
	database, err := db.New(mysql.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	ctx := t.Context()
	seeder := seed.New(t, database, nil)
	workspace := seeder.CreateWorkspace(ctx)
	projectID := uid.New(uid.ProjectPrefix)
	appID := uid.New(uid.AppPrefix)
	appMissingID := uid.New(uid.AppPrefix)
	appDeepOrphanID := uid.New(uid.AppPrefix)
	environmentID := uid.New(uid.EnvironmentPrefix)
	environmentMissingID := uid.New(uid.EnvironmentPrefix)
	environmentDeepOrphanID := uid.New(uid.EnvironmentPrefix)
	seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID: appID, WorkspaceID: workspace.ID, ProjectID: projectID, Name: "app", Slug: "app",
	})
	for _, environment := range []seed.CreateEnvironmentRequest{
		{ID: environmentID, WorkspaceID: workspace.ID, ProjectID: projectID, AppID: appID, Slug: "env"},
		{ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspace.ID, ProjectID: projectID, AppID: appMissingID, Slug: "env"},
	} {
		seeder.CreateEnvironment(ctx, environment)
	}
	for _, deployment := range []seed.CreateDeploymentRequest{
		{ProjectID: projectID, AppID: appID, EnvironmentID: environmentID},
		{ProjectID: projectID, AppID: appID, EnvironmentID: environmentMissingID},
		{ProjectID: projectID, AppID: appDeepOrphanID, EnvironmentID: environmentDeepOrphanID},
		{ProjectID: uid.New(uid.ProjectPrefix), AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix)},
	} {
		deployment.WorkspaceID = workspace.ID
		deployment.Status = mysqltype.DeploymentsStatusPending
		seeder.CreateDeployment(ctx, deployment)
	}
	apps, err := database.ListAppIdsByProject(ctx, db.ListAppIdsByProjectParams{ProjectID: projectID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{appID, appMissingID, appDeepOrphanID}, apps)
	environments, err := database.ListEnvironmentIdsByApp(ctx, db.ListEnvironmentIdsByAppParams{AppID: appID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{environmentID, environmentMissingID}, environments)
	environments, err = database.ListEnvironmentIdsByApp(ctx, db.ListEnvironmentIdsByAppParams{AppID: appDeepOrphanID})
	require.NoError(t, err)
	require.Equal(t, []string{environmentDeepOrphanID}, environments)
}
