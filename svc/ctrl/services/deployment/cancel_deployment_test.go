package deployment

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
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

// TestCancelDeploymentAuditsWithRequestActor pins the single-writer contract
// from the dedup-audit consolidation: the dashboard no longer inserts its own
// deployment.cancel entry, so the ctrl RPC must write it from the actor the
// request carries.
func TestCancelDeploymentAuditsWithRequestActor(t *testing.T) {
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

	actorID := uid.New("user")
	req := connect.NewRequest(&ctrlv1.CancelDeploymentRequest{
		DeploymentId: deployment.ID,
		Actor:        &ctrlv1.ActorInfo{Id: actorID, Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err := svc.CancelDeployment(ctx, req)
	require.NoError(t, err)

	require.Equal(t, 1, countCancelAudits(t, ctx, database, resources.workspaceID, deployment.ID, actorID))
}

// TestCancelDeploymentWithoutActorWritesNoAudit keeps out-of-band callers from
// fabricating a system-actor entry the customer never triggered.
func TestCancelDeploymentWithoutActorWritesNoAudit(t *testing.T) {
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

	req := connect.NewRequest(&ctrlv1.CancelDeploymentRequest{DeploymentId: deployment.ID})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err := svc.CancelDeployment(ctx, req)
	require.NoError(t, err)

	rows, err := database.ListClickhouseOutboxByWorkspace(ctx, resources.workspaceID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func countCancelAudits(t *testing.T, ctx context.Context, database db.Database, workspaceID, deploymentID, actorID string) int {
	t.Helper()
	rows, err := database.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		payload := string(row.Payload)
		if strings.Contains(payload, string(auditlog.DeploymentCancelEvent)) &&
			strings.Contains(payload, deploymentID) &&
			strings.Contains(payload, actorID) {
			count++
		}
	}
	return count
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

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	// RestateAdmin stays nil: these cases never reach the invocation cancel,
	// and a nil client is what proves it.
	svc := New(Config{
		Database:          database,
		Restate:           nil,
		RestateAdmin:      nil,
		Auditlogs:         auditlogSvc,
		Bearer:            testBearer,
		EnforceDeployGate: false,
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
