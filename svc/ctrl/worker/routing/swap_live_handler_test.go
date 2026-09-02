package routing_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
)

// TestSwapLiveDeploymentRefusesOlderUnlessAllowed covers the race between two
// concurrent production builds. Sibling dedup only supersedes deployments still
// queued (`deployment_list_older_active_for_dedup.sql` matches pending and
// awaiting_approval), so both builds run to completion and each swaps the live
// pointer, which would then belong to whichever finished last. Promote and
// rollback opt out, because confirm-rollback deliberately targets a deployment
// older than the live pointer.
func TestSwapLiveDeploymentRefusesOlderUnlessAllowed(t *testing.T) {
	t.Run("a build does not take traffic from a newer live deployment", func(t *testing.T) {
		h := newSwapFixture(t)

		// The newer commit built faster and is already serving.
		h.setLive(t, h.newer.ID)

		_, err := h.client.SwapLiveDeployment().Request(h.ctx, &hydrav1.SwapLiveDeploymentRequest{
			DeploymentId:      h.older.ID,
			FrontlineRouteIds: nil,
			SetRollbackFlag:   false,
			AllowOlder:        false,
		})
		require.NoError(t, err, "a refused swap is a no-op, not a failure")

		require.Equal(t, h.newer.ID, h.live(t),
			"the slower older build must not take traffic from the newer deployment")
	})

	t.Run("promote and rollback still move traffic to an older deployment", func(t *testing.T) {
		h := newSwapFixture(t)
		h.setLive(t, h.newer.ID)

		_, err := h.client.SwapLiveDeployment().Request(h.ctx, &hydrav1.SwapLiveDeploymentRequest{
			DeploymentId:      h.older.ID,
			FrontlineRouteIds: nil,
			SetRollbackFlag:   true,
			AllowOlder:        true,
		})
		require.NoError(t, err)

		require.Equal(t, h.older.ID, h.live(t),
			"an explicit rollback must be able to move traffic backwards")
	})

	t.Run("a build still takes traffic when it is the newest", func(t *testing.T) {
		h := newSwapFixture(t)
		h.setLive(t, h.older.ID)

		_, err := h.client.SwapLiveDeployment().Request(h.ctx, &hydrav1.SwapLiveDeploymentRequest{
			DeploymentId:      h.newer.ID,
			FrontlineRouteIds: nil,
			SetRollbackFlag:   false,
			AllowOlder:        false,
		})
		require.NoError(t, err)

		require.Equal(t, h.newer.ID, h.live(t),
			"the guard must only block a backwards swap")
	})

	t.Run("a build takes traffic when the app has no live deployment", func(t *testing.T) {
		h := newSwapFixture(t)

		_, err := h.client.SwapLiveDeployment().Request(h.ctx, &hydrav1.SwapLiveDeploymentRequest{
			DeploymentId:      h.older.ID,
			FrontlineRouteIds: nil,
			SetRollbackFlag:   false,
			AllowOlder:        false,
		})
		require.NoError(t, err)

		require.Equal(t, h.older.ID, h.live(t),
			"the first deployment of an app has nothing to be older than")
	})
}

type swapFixture struct {
	ctx    context.Context
	db     db.Database
	client hydrav1.RoutingServiceIngressClient
	appID  string
	older  db.Deployment
	newer  db.Deployment
}

// newSwapFixture builds one app in a production environment with two ready
// deployments a second apart. The tests drive the service through the ingress;
// production binds RoutingService ingress-private and calls it
// service-to-service.
func newSwapFixture(t *testing.T) *swapFixture {
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

	base := time.Now().UnixMilli()
	older := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     base,
	})
	newer := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     base + 1000,
	})

	restateCfg := containers.Restate(t, hydrav1.NewRoutingServiceServer(routing.New(routing.Config{
		DB:            database,
		DefaultDomain: "unkey.local",
	})))

	return &swapFixture{
		ctx:    ctx,
		db:     database,
		client: hydrav1.NewRoutingServiceIngressClient(restateCfg.IngressClient, environment.ID),
		appID:  app.ID,
		older:  older,
		newer:  newer,
	}
}

func (h *swapFixture) setLive(t *testing.T, deploymentID string) {
	t.Helper()
	require.NoError(t, h.db.UpdateAppDeployments(h.ctx, db.UpdateAppDeploymentsParams{
		AppID:               h.appID,
		CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))
}

func (h *swapFixture) live(t *testing.T) string {
	t.Helper()
	app, err := h.db.FindAppById(h.ctx, h.appID)
	require.NoError(t, err)
	return app.CurrentDeploymentID.String
}

func testSlug(prefix uid.Prefix) string {
	return strings.ToLower(strings.ReplaceAll(uid.New(prefix), "_", "-"))
}
