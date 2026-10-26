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
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
)

func TestBranchRouteSurvivesIdenticalLabelsFromAnotherWorkspace(t *testing.T) {
	ctx := context.Background()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)

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

	deployBranch := func(workspaceSlug, branch string) (db.Deployment, string) {
		workspaceID := uid.New("test_ws")
		require.NoError(t, database.InsertWorkspace(ctx, db.InsertWorkspaceParams{
			ID:           workspaceID,
			OrgID:        uid.New(uid.OrgPrefix),
			Name:         "KEBAP",
			Slug:         workspaceSlug,
			CreatedAt:    time.Now().UnixMilli(),
			K8sNamespace: uid.DNS1035(),
		}))
		project := seeder.CreateProject(ctx, seed.CreateProjectRequest{ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "KEBAP", Slug: "shop"})
		app := seeder.CreateApp(ctx, seed.CreateAppRequest{ID: uid.New(uid.AppPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, Name: "KEBAP", Slug: "web"})
		env := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
			ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID,
			Slug: "preview", Kind: mysqltype.EnvironmentKindPreview,
		})
		dep := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			ID: uid.New(uid.DeploymentPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID,
			EnvironmentID: env.ID, Status: mysqltype.DeploymentsStatusReady,
			GitBranch: sql.NullString{String: branch, Valid: true},
		})

		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := ingress.Workflow[string, RoutingProbeResult](cfg.IngressClient, "RoutingProbe", dep.ID, "Run").Request(callCtx, dep.ID)
		require.NoError(t, err)
		return dep, workspaceID
	}

	victimSlug := uid.DNS1035()
	victim, victimWorkspaceID := deployBranch(victimSlug, "feature-x")
	attacker, attackerWorkspaceID := deployBranch("x-"+victimSlug, "feature")

	victimRoute, err := database.FindFrontlineRouteByDeploymentIDAndSticky(ctx, db.FindFrontlineRouteByDeploymentIDAndStickyParams{
		DeploymentID: victim.ID,
		Sticky:       db.FrontlineRoutesStickyBranch,
	})
	require.NoError(t, err)

	attackerRoute, err := database.FindFrontlineRouteByDeploymentIDAndSticky(ctx, db.FindFrontlineRouteByDeploymentIDAndStickyParams{
		DeploymentID: attacker.ID,
		Sticky:       db.FrontlineRoutesStickyBranch,
	})
	require.NoError(t, err)

	t.Logf("victim ws:      %s  deployment %s", victimWorkspaceID, victim.ID)
	t.Logf("victim host:    %s", victimRoute.FullyQualifiedDomainName)
	t.Logf("attacker ws:    %s  deployment %s", attackerWorkspaceID, attacker.ID)
	t.Logf("attacker host:  %s", attackerRoute.FullyQualifiedDomainName)

	require.NotEqual(t, victimRoute.FullyQualifiedDomainName, attackerRoute.FullyQualifiedDomainName,
		"two workspaces that build the same bare label must get different hostnames")
	require.Equal(t, victim.EnvironmentID, victimRoute.EnvironmentID,
		"the victim's branch route must stay in the victim's environment")
	require.Equal(t, victim.ID, victimRoute.DeploymentID,
		"the victim's branch route must keep serving the victim deployment")
}
