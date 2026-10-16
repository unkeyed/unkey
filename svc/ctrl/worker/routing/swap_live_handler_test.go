package routing_test

import (
	"database/sql"
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

func TestSwapLiveDeployment_OlderCompletionCannotReplaceNewer(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	workspaceID := uid.New(uid.WorkspacePrefix)

	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New("prj"),
		WorkspaceID: workspaceID,
		Name:        "routing-race-project",
		Slug:        uid.New("slug"),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New("app"),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "routing-race-app",
		Slug:        "default",
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New("env"),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
		Description: "",
	})

	now := time.Now().UnixMilli()
	baseline := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     now,
	})
	older := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     now,
	})
	newer := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     now,
	})
	require.Less(t, baseline.Pk, older.Pk)
	require.Less(t, older.Pk, newer.Pk)

	require.NoError(t, database.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		AppID:               app.ID,
		CurrentDeploymentID: sql.NullString{Valid: true, String: baseline.ID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
	}))

	routeID := uid.New("fr")
	fqdn := uid.DNS1035() + ".example.com"
	require.NoError(t, database.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       routeID,
		ProjectID:                project.ID,
		AppID:                    app.ID,
		DeploymentID:             baseline.ID,
		EnvironmentID:            environment.ID,
		FullyQualifiedDomainName: fqdn,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Valid: false},
	}))

	lateDomain := uid.DNS1035() + ".example.com"
	require.NoError(t, database.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                project.ID,
		AppID:                    app.ID,
		DeploymentID:             baseline.ID,
		EnvironmentID:            environment.ID,
		FullyQualifiedDomainName: lateDomain,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                now,
	}))

	tEnv := containers.Restate(t, hydrav1.NewRoutingServiceServer(routing.New(routing.Config{
		DB:            database,
		DefaultDomain: "example.com",
	})))
	client := hydrav1.NewRoutingServiceIngressClient(tEnv.IngressClient, environment.ID)

	newerResponse, err := client.SwapLiveDeployment().Request(ctx, &hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:       newer.ID,
		FrontlineRouteIds:  []string{routeID},
		AutomaticPromotion: true,
	})
	require.NoError(t, err)
	require.Equal(t,
		hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_UNSPECIFIED,
		newerResponse.GetAutomaticPromotionSkipReason(),
	)
	require.Equal(t, baseline.ID, newerResponse.GetPreviousDeploymentId())

	duplicateResponse, err := client.SwapLiveDeployment().Request(ctx, &hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:       newer.ID,
		FrontlineRouteIds:  []string{routeID},
		AutomaticPromotion: true,
	})
	require.NoError(t, err)
	require.Equal(t,
		hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_UNSPECIFIED,
		duplicateResponse.GetAutomaticPromotionSkipReason(),
	)
	require.Empty(t, duplicateResponse.GetPreviousDeploymentId())

	olderResponse, err := client.SwapLiveDeployment().Request(ctx, &hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:       older.ID,
		FrontlineRouteIds:  []string{routeID},
		AutomaticPromotion: true,
	})
	require.NoError(t, err)
	require.Equal(t,
		hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_NEWER_DEPLOYMENT,
		olderResponse.GetAutomaticPromotionSkipReason(),
	)
	require.Empty(t, olderResponse.GetPreviousDeploymentId())

	gotApp, err := database.FindAppById(ctx, app.ID)
	require.NoError(t, err)
	require.True(t, gotApp.CurrentDeploymentID.Valid)
	require.Equal(t, newer.ID, gotApp.CurrentDeploymentID.String)

	gotRoute, err := database.FindFrontlineRouteByFQDN(ctx, fqdn)
	require.NoError(t, err)
	require.Equal(t, newer.ID, gotRoute.DeploymentID)
	lateRoute, err := database.FindFrontlineRouteByFQDN(ctx, lateDomain)
	require.NoError(t, err)
	require.Equal(t, newer.ID, lateRoute.DeploymentID, "routes added after the caller collected route IDs must also be promoted")

	rollbackResponse, err := client.SwapLiveDeployment().Request(ctx, &hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:      baseline.ID,
		FrontlineRouteIds: []string{routeID},
		SetRollbackFlag:   true,
	})
	require.NoError(t, err)
	require.Equal(t, newer.ID, rollbackResponse.GetPreviousDeploymentId())

	afterRollback := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     now,
	})
	require.Greater(t, afterRollback.Pk, newer.Pk)

	afterRollbackResponse, err := client.SwapLiveDeployment().Request(ctx, &hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:       afterRollback.ID,
		FrontlineRouteIds:  []string{routeID},
		AutomaticPromotion: true,
	})
	require.NoError(t, err)
	require.Equal(t,
		hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_ROLLED_BACK,
		afterRollbackResponse.GetAutomaticPromotionSkipReason(),
	)

	gotApp, err = database.FindAppById(ctx, app.ID)
	require.NoError(t, err)
	require.Equal(t, baseline.ID, gotApp.CurrentDeploymentID.String)
	require.True(t, gotApp.IsRolledBack)

	gotRoute, err = database.FindFrontlineRouteByFQDN(ctx, fqdn)
	require.NoError(t, err)
	require.Equal(t, baseline.ID, gotRoute.DeploymentID)
	lateRoute, err = database.FindFrontlineRouteByFQDN(ctx, lateDomain)
	require.NoError(t, err)
	require.Equal(t, baseline.ID, lateRoute.DeploymentID)
}
