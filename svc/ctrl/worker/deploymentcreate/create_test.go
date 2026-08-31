package deploymentcreate

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestInsertDeploymentRecordsQueuedStep pins the queued step to the insert
// transaction. Deploy's Dequeue ends the queued step by (deployment, step)
// with no insert of its own, so a row created without it shows no progress in
// the UI until the Starting step lands.
func TestInsertDeploymentRecordsQueuedStep(t *testing.T) {
	ctx := context.Background()
	svc, target := newTestService(t, ctx)

	deploymentID := uid.New(uid.DeploymentPrefix)
	out, err := svc.insertDeployment(ctx, &hydrav1.DeploymentCreateRequest{
		ProjectId:       target.projectID,
		AppId:           target.appID,
		EnvironmentSlug: target.environmentSlug,
		DeployRequest:   &hydrav1.DeployRequest{DeploymentId: deploymentID},
		Trigger:         ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API,
		Action:          hydrav1.DeploymentCreateAction_DEPLOYMENT_CREATE_ACTION_CREATE,
	}, deploymentID)
	require.NoError(t, err)
	require.Equal(t, target.environmentID, out.EnvironmentID)

	deployment, err := svc.db.FindDeploymentById(ctx, deploymentID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusPending, deployment.Status)

	var (
		step    string
		endedAt *int64
	)
	err = svc.db.RO().QueryRowContext(ctx,
		"SELECT step, ended_at FROM deployment_steps WHERE deployment_id = ?",
		deploymentID,
	).Scan(&step, &endedAt)
	require.NoError(t, err, "the insert transaction must record the queued step")
	require.Equal(t, string(db.DeploymentStepsStepQueued), step)
	require.Nil(t, endedAt, "the queued step stays open until Deploy dequeues it")
}

// testTarget is the seeded (project, app, environment) triple insertDeployment
// resolves through deploytarget.Load.
type testTarget struct {
	projectID       string
	appID           string
	environmentID   string
	environmentSlug string
	workspaceID     string
}

func newTestService(t *testing.T, ctx context.Context) (*Service, testTarget) {
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
		Slug:        slug(uid.ProjectPrefix),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "KEBAP",
		Slug:          slug(uid.AppPrefix),
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

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	return New(Config{DB: database, Auditlogs: auditlogSvc, RestateAdmin: nil}), testTarget{
		projectID:       project.ID,
		appID:           app.ID,
		environmentID:   environment.ID,
		environmentSlug: environment.Slug,
		workspaceID:     workspaceID,
	}
}

func slug(prefix uid.Prefix) string {
	return strings.ToLower(strings.ReplaceAll(uid.New(prefix), "_", "-"))
}
