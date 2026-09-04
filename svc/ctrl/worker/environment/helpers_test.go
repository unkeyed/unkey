package environment_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/worker/environment"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
)

// stopDelay is how long a test-scheduled stop waits before it applies. Long
// enough for the handler under test to run first, short enough to wait out.
const stopDelay = 300 * time.Millisecond

// fixture is one production environment with three ready deployments.
// live owns the sticky routes and a commit route; the others are candidates.
// The seeder has no helper for routes or the app's live pointer, so the fixture
// writes those rows itself.
type fixture struct {
	ctx         context.Context
	db          db.Database
	seeder      *seed.Seeder
	restate     containers.RestateConfig
	workspaceID string
	project     db.Project
	app         db.App
	env         db.Environment
	live        db.Deployment
	candidate   db.Deployment
	other       db.Deployment
	actorID     string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
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
		Slug:        slugFrom(uid.New(uid.ProjectPrefix)),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "KEBAP",
		Slug:          slugFrom(uid.New(uid.AppPrefix)),
		DefaultBranch: "main",
	})

	f := &fixture{
		ctx:         ctx,
		db:          database,
		seeder:      seeder,
		restate:     containers.RestateConfig{},
		workspaceID: workspaceID,
		project:     project,
		app:         app,
		env:         db.Environment{},
		live:        db.Deployment{},
		candidate:   db.Deployment{},
		other:       db.Deployment{},
		actorID:     uid.New("user"),
	}
	f.env = f.newProductionEnvironment(t, "production")
	f.live = f.newReadyDeployment(t)
	f.candidate = f.newReadyDeployment(t)
	f.other = f.newReadyDeployment(t)

	f.insertRoute(t, f.live.ID, db.FrontlineRoutesStickyLive)
	f.insertRoute(t, f.live.ID, db.FrontlineRoutesStickyEnvironment)
	f.insertRoute(t, f.live.ID, db.FrontlineRoutesStickyNone)
	f.setLive(t, f.live.ID, false)

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	var svc *environment.Service
	f.restate = containers.Restate(t,
		hydrav1.NewEnvironmentServiceServer(&lazyEnvironmentService{svc: &svc}),
		hydrav1.NewRoutingServiceServer(routing.New(routing.Config{DB: database, DefaultDomain: "kebap.test"})),
		hydrav1.NewDeploymentServiceServer(deployment.New(deployment.Config{DB: database})),
	)
	svc, err = environment.New(environment.Config{
		DB:        database,
		Admin:     restateadmin.New(restateadmin.Config{BaseURL: f.restate.AdminURL, APIKey: ""}),
		Auditlogs: auditlogSvc,
	})
	require.NoError(t, err)

	return f
}

func (f *fixture) newProductionEnvironment(t *testing.T, slug string) db.Environment {
	t.Helper()
	return f.seeder.CreateEnvironment(f.ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: f.workspaceID,
		ProjectID:   f.project.ID,
		AppID:       f.app.ID,
		Slug:        slug,
		Kind:        mysqltype.EnvironmentKindProduction,
	})
}

func (f *fixture) newReadyDeployment(t *testing.T) db.Deployment {
	t.Helper()
	return f.seeder.CreateDeployment(f.ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.project.ID,
		AppID:         f.app.ID,
		EnvironmentID: f.env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
}

func (f *fixture) insertRoute(t *testing.T, deploymentID string, sticky db.FrontlineRoutesSticky) {
	t.Helper()
	require.NoError(t, f.db.InsertFrontlineRoute(f.ctx, db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                f.project.ID,
		AppID:                    f.app.ID,
		DeploymentID:             deploymentID,
		EnvironmentID:            f.env.ID,
		FullyQualifiedDomainName: slugFrom(uid.New(uid.FrontlineRoutePrefix)) + ".kebap.test",
		Sticky:                   sticky,
		CreatedAt:                time.Now().UnixMilli(),
		UpdatedAt:                sql.NullInt64{Valid: false, Int64: 0},
	}))
}

// moveStickyRoutes points the live and environment routes at deploymentID.
func (f *fixture) moveStickyRoutes(t *testing.T, deploymentID string) {
	t.Helper()
	routes, err := f.db.FindFrontlineRoutesByEnvironmentAndSticky(f.ctx, db.FindFrontlineRoutesByEnvironmentAndStickyParams{
		EnvironmentID: f.env.ID,
		Sticky:        []db.FrontlineRoutesSticky{db.FrontlineRoutesStickyLive, db.FrontlineRoutesStickyEnvironment},
	})
	require.NoError(t, err)
	for _, route := range routes {
		require.NoError(t, f.db.ReassignFrontlineRoute(f.ctx, db.ReassignFrontlineRouteParams{
			DeploymentID: deploymentID,
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ID:           route.ID,
		}))
	}
}

func (f *fixture) setLive(t *testing.T, deploymentID string, rolledBack bool) {
	t.Helper()
	require.NoError(t, f.db.UpdateAppDeployments(f.ctx, db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
		IsRolledBack:        rolledBack,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		AppID:               f.app.ID,
	}))
}

func (f *fixture) actor() *ctrlv1.ActorInfo {
	return &ctrlv1.ActorInfo{Id: f.actorID, Type: ctrlv1.ActorType_ACTOR_TYPE_USER}
}

func (f *fixture) promote(key, deploymentID string) error {
	_, err := hydrav1.NewEnvironmentServiceIngressClient(f.restate.IngressClient, key).
		PromoteDeployment().
		Request(f.ctx, &hydrav1.PromoteDeploymentRequest{
			DeploymentId:  deploymentID,
			Actor:         f.actor(),
			CorrelationId: uid.New("corr"),
		})
	return err
}

func (f *fixture) rollback(key, fromID, toID string) error {
	_, err := hydrav1.NewEnvironmentServiceIngressClient(f.restate.IngressClient, key).
		RollbackDeployment().
		Request(f.ctx, &hydrav1.RollbackDeploymentRequest{
			FromDeploymentId: fromID,
			ToDeploymentId:   toID,
			Actor:            f.actor(),
			CorrelationId:    uid.New("corr"),
		})
	return err
}

// scheduleStop parks a stop on deploymentID that lands after stopDelay. A
// handler that clears it keeps the deployment running; one that does not lets
// it flip to stopped.
func (f *fixture) scheduleStop(t *testing.T, deploymentID string) {
	t.Helper()
	_, err := hydrav1.NewDeploymentServiceIngressClient(f.restate.IngressClient, deploymentID).
		ScheduleDesiredStateChange().
		Request(f.ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
			State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
			DelayMillis: stopDelay.Milliseconds(),
			Overwrite:   true,
		})
	require.NoError(t, err)
}

// requireDesiredStateAfterStopDelay waits out stopDelay so a parked stop had
// its chance, then asserts.
func (f *fixture) requireDesiredStateAfterStopDelay(t *testing.T, deploymentID string, want mysqltype.DeploymentsDesiredState) {
	t.Helper()
	time.Sleep(3 * stopDelay)
	dep, err := f.db.FindDeploymentById(f.ctx, deploymentID)
	require.NoError(t, err)
	require.Equal(t, want, dep.DesiredState, "deployment %s desired state", deploymentID)
}

func (f *fixture) requireLive(t *testing.T, deploymentID string, rolledBack bool) {
	t.Helper()
	app, err := f.db.FindAppById(f.ctx, f.app.ID)
	require.NoError(t, err)
	require.Equal(t, deploymentID, app.CurrentDeploymentID.String, "app current_deployment_id")
	require.Equal(t, rolledBack, app.IsRolledBack, "app is_rolled_back")
}

// routeOwners maps each route kind in the environment to the deployment it
// points at.
func (f *fixture) routeOwners(t *testing.T) map[db.FrontlineRoutesSticky]string {
	t.Helper()
	routes, err := f.db.FindFrontlineRoutesByEnvironmentAndSticky(f.ctx, db.FindFrontlineRoutesByEnvironmentAndStickyParams{
		EnvironmentID: f.env.ID,
		Sticky: []db.FrontlineRoutesSticky{
			db.FrontlineRoutesStickyLive,
			db.FrontlineRoutesStickyEnvironment,
			db.FrontlineRoutesStickyNone,
		},
	})
	require.NoError(t, err)
	require.Len(t, routes, 3)

	owners := make(map[db.FrontlineRoutesSticky]string, len(routes))
	for _, route := range routes {
		owners[route.Sticky] = route.DeploymentID
	}
	return owners
}

func (f *fixture) requireRoutes(t *testing.T, stickyOwner, commitOwner string) {
	t.Helper()
	owners := f.routeOwners(t)
	require.Equal(t, stickyOwner, owners[db.FrontlineRoutesStickyLive], "live route")
	require.Equal(t, stickyOwner, owners[db.FrontlineRoutesStickyEnvironment], "environment route")
	require.Equal(t, commitOwner, owners[db.FrontlineRoutesStickyNone], "commit route")
}

func slugFrom(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "_", "-"))
}
