//go:build integration

package integration

import (
	"database/sql"
	"testing"

	restatetest "github.com/restatedev/sdk-go/testing"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
)

// TestSwapLiveDeployment_OlderCompletionCannotReplaceNewer verifies that the
// environment-serialized routing transaction keeps both the live pointer and
// sticky routes on the newest deployment when builds finish out of order.
func TestSwapLiveDeployment_OlderCompletionCannotReplaceNewer(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	workspaceID := h.Resources().UserWorkspace.ID

	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New("prj"),
		WorkspaceID: workspaceID,
		Name:        "routing-race-project",
		Slug:        uid.New("slug"),
	})
	app := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "routing-race-app",
		Slug:          "default",
		DefaultBranch: "main",
	})
	environment := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:             uid.New("env"),
		WorkspaceID:    workspaceID,
		ProjectID:      project.ID,
		AppID:          app.ID,
		Slug:           "production",
		Kind:           mysqltype.EnvironmentKindProduction,
		Description:    "",
		SentinelConfig: []byte("{}"),
	})

	// All deployments intentionally share the same millisecond timestamp. Their
	// auto-increment primary keys must still provide a strict creation order.
	now := h.Now()
	baseline := h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     now,
	})
	older := h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     now,
	})
	newer := h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     now,
	})
	require.Less(t, baseline.Pk, older.Pk)
	require.Less(t, older.Pk, newer.Pk)

	require.NoError(t, h.DB.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		AppID:               app.ID,
		CurrentDeploymentID: sql.NullString{Valid: true, String: baseline.ID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
	}))

	routeID := uid.New("fr")
	fqdn := uid.DNS1035() + ".example.com"
	require.NoError(t, h.DB.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
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

	tEnv := restatetest.Start(t, hydrav1.NewRoutingServiceServer(routing.New(routing.Config{
		DB:            h.DB,
		DefaultDomain: "example.com",
	})))
	client := hydrav1.NewRoutingServiceIngressClient(tEnv.Ingress(), environment.ID)

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

	// A replay after an ambiguous commit must not report the live target as its
	// own previous deployment, which would schedule it for standby.
	replayedResponse, err := client.SwapLiveDeployment().Request(ctx, &hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:       newer.ID,
		FrontlineRouteIds:  []string{routeID},
		AutomaticPromotion: true,
	})
	require.NoError(t, err)
	require.Equal(t,
		hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_UNSPECIFIED,
		replayedResponse.GetAutomaticPromotionSkipReason(),
	)
	require.Empty(t, replayedResponse.GetPreviousDeploymentId())

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

	gotApp, err := h.DB.FindAppById(ctx, app.ID)
	require.NoError(t, err)
	require.True(t, gotApp.CurrentDeploymentID.Valid)
	require.Equal(t, newer.ID, gotApp.CurrentDeploymentID.String)

	gotRoute, err := h.DB.FindFrontlineRouteByFQDN(ctx, fqdn)
	require.NoError(t, err)
	require.Equal(t, newer.ID, gotRoute.DeploymentID)

	// A rollback that wins environment serialization must remain sticky even
	// when a newer automatic deployment finishes afterwards.
	rollbackResponse, err := client.SwapLiveDeployment().Request(ctx, &hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:      baseline.ID,
		FrontlineRouteIds: []string{routeID},
		SetRollbackFlag:   true,
	})
	require.NoError(t, err)
	require.Equal(t, newer.ID, rollbackResponse.GetPreviousDeploymentId())

	afterRollback := h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
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

	gotApp, err = h.DB.FindAppById(ctx, app.ID)
	require.NoError(t, err)
	require.Equal(t, baseline.ID, gotApp.CurrentDeploymentID.String)
	require.True(t, gotApp.IsRolledBack)

	gotRoute, err = h.DB.FindFrontlineRouteByFQDN(ctx, fqdn)
	require.NoError(t, err)
	require.Equal(t, baseline.ID, gotRoute.DeploymentID)
}
