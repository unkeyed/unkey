package deployment

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestCancelDeploymentWithoutInvocationID covers the window between the create
// sending Deploy and persisting the invocation id: the column still reads NULL,
// there is nothing to cancel in Restate, and a success that left the row pending
// would let the build run to completion after the user cancelled it.
func TestCancelDeploymentWithoutInvocationID(t *testing.T) {
	ctx := context.Background()
	svc, database, resources := newCancelTestService(t, ctx)

	deployment := resources.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   resources.workspaceID,
		ProjectID:     resources.projectID,
		AppID:         resources.appID,
		EnvironmentID: resources.environmentID,
		Status:        mysqltype.DeploymentsStatusPending,
	})
	require.False(t, deployment.InvocationID.Valid)

	req := connect.NewRequest(&ctrlv1.CancelDeploymentRequest{DeploymentId: deployment.ID})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err := svc.CancelDeployment(ctx, req)
	require.NoError(t, err)

	cancelled, err := database.FindDeploymentById(ctx, deployment.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusCancelled, cancelled.Status,
		"a cancel arriving before the invocation id lands must still stop the deployment")
}

// TestCancelDeploymentTerminalIsNoop pins the existing short-circuit: a
// deployment that already finished is left exactly as it is.
func TestCancelDeploymentTerminalIsNoop(t *testing.T) {
	ctx := context.Background()
	svc, database, resources := newCancelTestService(t, ctx)

	deployment := resources.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   resources.workspaceID,
		ProjectID:     resources.projectID,
		AppID:         resources.appID,
		EnvironmentID: resources.environmentID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	req := connect.NewRequest(&ctrlv1.CancelDeploymentRequest{DeploymentId: deployment.ID})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err := svc.CancelDeployment(ctx, req)
	require.NoError(t, err)

	after, err := database.FindDeploymentById(ctx, deployment.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusReady, after.Status)
}

const testBearer = "KEBAP"

// cancelTestResources is the seeded hierarchy a deployment row needs.
type cancelTestResources struct {
	seeder        *seed.Seeder
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func newCancelTestService(t *testing.T, ctx context.Context) (*Service, db.Database, cancelTestResources) {
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
		Slug:        testSlug(uid.ProjectPrefix),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "KEBAP",
		Slug:          testSlug(uid.AppPrefix),
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

	// RestateAdmin stays nil: these cases never reach the invocation cancel,
	// and a nil client is what proves it.
	svc := New(Config{
		Database:                        database,
		Restate:                         nil,
		RestateAdmin:                    nil,
		GitHub:                          nil,
		Auditlogs:                       nil,
		AllowUnauthenticatedDeployments: false,
		Bearer:                          testBearer,
		EnforceDeployGate:               false,
	})

	return svc, database, cancelTestResources{
		seeder:        seeder,
		workspaceID:   workspaceID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

func testSlug(prefix uid.Prefix) string {
	return strings.ToLower(strings.ReplaceAll(uid.New(prefix), "_", "-"))
}
