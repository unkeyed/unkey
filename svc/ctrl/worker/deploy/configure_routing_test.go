package deploy_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/internal/services/flags"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
)

// TestConfigureRoutingKeepsLiveRouteOnRolledBackApp pins that a new production
// deployment of a rolled-back app moves its environment and branch routes but
// leaves the live route on the deployment the app was rolled back to.
func TestConfigureRoutingKeepsLiveRouteOnRolledBackApp(t *testing.T) {
	ctx := context.Background()
	database, fixture := newDeployFixture(t, ctx)

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	workflow, err := deploy.New(deploy.Config{
		DB:            database,
		Auditlogs:     auditlogSvc,
		Flags:         flags.NewNoop(),
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.local",
		Vault:         nil,
		GitHub:        githubclient.NewNoop(),
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
		RestateAdmin:                    nil,
	})
	require.NoError(t, err)

	cfg := containers.Restate(t,
		restate.Reflect(RoutingProbe{workflow: workflow, db: database}),
		hydrav1.NewRoutingServiceServer(routing.New(routing.Config{DB: database, DefaultDomain: "test.example.com"})),
	)

	configureRouting := func(t *testing.T, deploymentID string) RoutingProbeResult {
		t.Helper()
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		result, err := ingress.Workflow[string, RoutingProbeResult](cfg.IngressClient, "RoutingProbe", deploymentID, "Run").
			Request(callCtx, deploymentID)
		require.NoError(t, err)
		return result
	}

	createMainDeployment := func() db.Deployment {
		return fixture.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   fixture.workspaceID,
			ProjectID:     fixture.projectID,
			AppID:         fixture.appID,
			EnvironmentID: fixture.environmentID,
			Status:        mysqltype.DeploymentsStatusReady,
			GitBranch:     sql.NullString{String: "main", Valid: true},
		})
	}

	routeDeployment := func(t *testing.T, sticky db.FrontlineRoutesSticky) db.FindFrontlineRoutesByEnvironmentAndStickyRow {
		t.Helper()
		routes, err := database.FindFrontlineRoutesByEnvironmentAndSticky(ctx, db.FindFrontlineRoutesByEnvironmentAndStickyParams{
			EnvironmentID: fixture.environmentID,
			Sticky:        []db.FrontlineRoutesSticky{sticky},
		})
		require.NoError(t, err)
		require.Len(t, routes, 1, "expected one %s route", sticky)
		return routes[0]
	}

	rolledBackTo := createMainDeployment()
	initial := configureRouting(t, rolledBackTo.ID)
	require.True(t, initial.ShouldAutoPromote, "the first production deployment of a live app promotes")
	require.Contains(t, initial.ReassignRouteIDs, routeDeployment(t, db.FrontlineRoutesStickyLive).ID,
		"a production deployment of an app that is not rolled back promotes the live route")

	require.NoError(t, database.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: rolledBackTo.ID, Valid: true},
		IsRolledBack:        true,
		UpdatedAt:           sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		AppID:               fixture.appID,
	}))

	next := createMainDeployment()
	result := configureRouting(t, next.ID)
	require.False(t, result.ShouldAutoPromote, "a rolled-back app must not auto promote")

	require.Equal(t, rolledBackTo.ID, routeDeployment(t, db.FrontlineRoutesStickyLive).DeploymentID,
		"the live route must stay on the rolled-back deployment")
	require.Equal(t, next.ID, routeDeployment(t, db.FrontlineRoutesStickyEnvironment).DeploymentID,
		"the environment route follows the newest deployment")
	require.Equal(t, next.ID, routeDeployment(t, db.FrontlineRoutesStickyBranch).DeploymentID,
		"the branch route follows the newest deployment of its branch")
}

type RoutingProbeResult struct {
	ReassignRouteIDs  []string `json:"reassignRouteIds"`
	ShouldAutoPromote bool     `json:"shouldAutoPromote"`
}

// RoutingProbe hosts ConfigureRouting behind a Restate workflow so the test can
// hand it a real WorkflowContext.
type RoutingProbe struct {
	workflow *deploy.Workflow
	db       db.Database
}

func (p RoutingProbe) Run(ctx restate.WorkflowContext, deploymentID string) (RoutingProbeResult, error) {
	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindDeploymentForDeployRow, error) {
		return p.db.FindDeploymentForDeploy(runCtx, deploymentID)
	})
	if err != nil {
		return RoutingProbeResult{}, err
	}

	reassignRouteIDs, shouldAutoPromote, routingErr := p.workflow.ConfigureRouting(ctx, deployment)
	if routingErr != nil {
		return RoutingProbeResult{}, routingErr
	}
	return RoutingProbeResult{ReassignRouteIDs: reassignRouteIDs, ShouldAutoPromote: shouldAutoPromote}, nil
}
