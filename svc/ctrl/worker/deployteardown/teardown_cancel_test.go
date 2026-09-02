package deployteardown_test

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

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
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployteardown"
)

// TestTeardownCancelsInFlightDeploy pins the cancel-first contract: only a
// deployment still progressing when its workspace is torn down gets its Deploy
// invocation cancelled. Without the cancel, that build keeps provisioning
// compute for a workspace that must not run any.
func TestTeardownCancelsInFlightDeploy(t *testing.T) {
	ctx := context.Background()

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
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New(uid.ProjectPrefix), "_", "-")),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "KEBAP",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New(uid.AppPrefix), "_", "-")),
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
		Kind:        mysqltype.EnvironmentKindPreview,
	})

	seedDeployment := func(status mysqltype.DeploymentsStatus) (db.Deployment, string) {
		dep := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			WorkspaceID:   workspaceID,
			ProjectID:     project.ID,
			AppID:         app.ID,
			EnvironmentID: environment.ID,
			Status:        status,
		})
		invocationID := uid.New("inv")
		require.NoError(t, database.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
			InvocationID: sql.NullString{Valid: true, String: invocationID},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ID:           dep.ID,
		}))
		return dep, invocationID
	}

	building, buildingInvocation := seedDeployment(mysqltype.DeploymentsStatusBuilding)
	ready, _ := seedDeployment(mysqltype.DeploymentsStatusReady)

	recorder := &recordingCanceler{}

	teardownSvc, err := deployteardown.New(deployteardown.Config{
		DB:                database,
		Admin:             recorder,
		DrainPollInterval: 200 * time.Millisecond,
		DrainGraceTimeout: 2 * time.Second,
	})
	require.NoError(t, err)

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	// Teardown sends ScheduleDesiredStateChange to DeployService, so the real
	// workflow must be bound or the sends would retry against nothing.
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

	restateCfg := containers.Restate(t,
		hydrav1.NewDeployTeardownServiceServer(teardownSvc),
		hydrav1.NewDeployServiceServer(workflow),
	)

	resp, err := hydrav1.NewDeployTeardownServiceIngressClient(restateCfg.IngressClient, workspaceID).
		Teardown().
		Request(ctx, &hydrav1.TeardownRequest{Mode: hydrav1.TeardownMode_TEARDOWN_MODE_ARCHIVE})
	require.NoError(t, err)
	require.Equal(t, int32(2), resp.GetDeploymentsStopped())

	require.Equal(t, []string{buildingInvocation}, recorder.calls(),
		"exactly the progressing deployment's invocation must be cancelled")

	// The stops are sent, not requested, so the rows only reach desired_state
	// stopped once DeployService has processed them.
	require.Eventually(t, func() bool {
		buildingAfter, findErr := database.FindDeploymentById(ctx, building.ID)
		if findErr != nil {
			return false
		}
		readyAfter, findErr := database.FindDeploymentById(ctx, ready.ID)
		if findErr != nil {
			return false
		}
		return buildingAfter.DesiredState == mysqltype.DeploymentsDesiredStateStopped &&
			readyAfter.DesiredState == mysqltype.DeploymentsDesiredStateStopped
	}, 15*time.Second, 200*time.Millisecond, "both deployments must reach desired_state=stopped")
}

// recordingCanceler records cancelled invocation ids instead of talking to a
// Restate admin API.
type recordingCanceler struct {
	mu  sync.Mutex
	ids []string
}

func (r *recordingCanceler) CancelInvocation(_ context.Context, invocationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ids = append(r.ids, invocationID)
	return nil
}

func (r *recordingCanceler) calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.ids...)
}
