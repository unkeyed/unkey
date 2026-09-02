package deploy_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
)

// TestDeployStopsOnTerminalStatus covers a cancel that lands before the
// invocation id does: the row is already cancelled and there is no invocation
// to cancel, so Deploy itself has to refuse to build. The workflow here has no
// GitHub client, no registry, and no cluster, so any step past the guard fails
// loudly.
func TestDeployStopsOnTerminalStatus(t *testing.T) {
	ctx := context.Background()
	database, resources := newDeployFixture(t, ctx)

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	workflow, err := deploy.New(deploy.Config{
		DB:            database,
		Auditlogs:     auditlogSvc,
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.com",
		Vault:         nil,
		GitHub:        nil,
		Build: deploy.BuildConfig{
			Backend:    deploy.BuildBackendDepot,
			Depot:      deploy.DepotConfig{APIUrl: "", ProjectRegion: "", ProjectPrefix: "builds-test"},
			Kubernetes: deploy.KubernetesBuildConfig{Namespace: "", Image: ""},
		},
		K8s:                             nil,
		RegistryConfig:                  deploy.RegistryConfig{Repository: "", Username: "", Password: "", Insecure: false},
		BuildPlatform:                   deploy.BuildPlatform{Platform: "", Architecture: ""},
		Clickhouse:                      nil,
		BuildSteps:                      batch.NewNoop[schema.BuildStepV1](),
		BuildStepLogs:                   batch.NewNoop[schema.BuildStepLogV1](),
		AllowUnauthenticatedDeployments: false,
	})
	require.NoError(t, err)

	restateCfg := containers.Restate(t, hydrav1.NewDeployServiceServer(workflow))

	deployment := resources.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   resources.workspaceID,
		ProjectID:     resources.projectID,
		AppID:         resources.appID,
		EnvironmentID: resources.environmentID,
		Status:        mysqltype.DeploymentsStatusCancelled,
		GitBranch:     sql.NullString{Valid: true, String: "main"},
	})

	_, err = hydrav1.NewDeployServiceIngressClient(restateCfg.IngressClient, deployment.ID).
		Deploy().
		Request(ctx, &hydrav1.DeployRequest{DeploymentId: deployment.ID})
	require.NoError(t, err)

	after, err := database.FindDeploymentById(ctx, deployment.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusCancelled, after.Status,
		"the cancelled status must survive the Deploy invocation")

	// The guard runs after the invocation id write, so the row can be cancelled
	// through Restate admin even when the cancel raced the create.
	require.True(t, after.InvocationID.Valid, "Deploy re-persists its own invocation id")
	require.NotEmpty(t, after.InvocationID.String)

	var steps int
	require.NoError(t, database.RO().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM deployment_steps WHERE deployment_id = ?", deployment.ID,
	).Scan(&steps))
	require.Zero(t, steps, "a terminal row must not start a deployment step")
}

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
