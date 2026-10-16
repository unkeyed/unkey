package deploy

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestFrontlineRouteRegistration(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: uid.New(uid.WorkspacePrefix),
		ProjectID:   uid.New(uid.ProjectPrefix),
		Name:        "routing",
		Slug:        "routing",
	})
	w := &Workflow{db: database}
	deployment := db.FindDeploymentForDeployRow{
		ID:              uid.New(uid.DeploymentPrefix),
		AppID:           app.ID,
		ProjectID:       app.ProjectID,
		EnvironmentID:   uid.New(uid.EnvironmentPrefix),
		EnvironmentKind: mysqltype.EnvironmentKindProduction,
	}

	t.Run("concurrent registrations share one route", func(t *testing.T) {
		domain := newDomain{domain: uid.DNS1035() + ".example.com", sticky: db.FrontlineRoutesStickyLive}
		other := deployment
		other.ID = uid.New(uid.DeploymentPrefix)
		deployments := []db.FindDeploymentForDeployRow{deployment, other}
		routeIDs := make([]string, len(deployments))
		errs := make([]error, len(deployments))
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i, candidate := range deployments {
			wg.Go(func() {
				<-start
				routeIDs[i], errs[i] = w.createFrontlineRoute(ctx, candidate, domain)
			})
		}
		close(start)
		wg.Wait()
		for _, err := range errs {
			require.NoError(t, err)
		}
		require.NotEmpty(t, routeIDs[0])
		require.Equal(t, routeIDs[0], routeIDs[1])
		route, err := database.FindFrontlineRouteByFQDN(ctx, domain.domain)
		require.NoError(t, err)
		require.Equal(t, routeIDs[0], route.ID)
		require.Contains(t, []string{deployment.ID, other.ID}, route.DeploymentID)
		require.Equal(t, app.ID, route.AppID)
		require.Equal(t, deployment.EnvironmentID, route.EnvironmentID)
	})

	t.Run("conflicting ownership or policy fails without changing the route", func(t *testing.T) {
		for _, field := range []string{"project", "app", "environment", "sticky"} {
			t.Run(field, func(t *testing.T) {
				domain := newDomain{domain: uid.DNS1035() + ".example.com", sticky: db.FrontlineRoutesStickyLive}
				previous := db.InsertFrontlineRouteParams{
					ID:                       uid.New(uid.FrontlineRoutePrefix),
					ProjectID:                deployment.ProjectID,
					AppID:                    deployment.AppID,
					EnvironmentID:            deployment.EnvironmentID,
					DeploymentID:             uid.New(uid.DeploymentPrefix),
					FullyQualifiedDomainName: domain.domain,
					Sticky:                   domain.sticky,
					CreatedAt:                time.Now().UnixMilli(),
				}
				switch field {
				case "project":
					previous.ProjectID = uid.New(uid.ProjectPrefix)
				case "app":
					previous.AppID = uid.New(uid.AppPrefix)
				case "environment":
					previous.EnvironmentID = uid.New(uid.EnvironmentPrefix)
				case "sticky":
					previous.Sticky = db.FrontlineRoutesStickyBranch
				}
				require.NoError(t, database.InsertFrontlineRoute(ctx, previous))
				before, err := database.FindFrontlineRouteByFQDN(ctx, domain.domain)
				require.NoError(t, err)
				_, err = w.createFrontlineRoute(ctx, deployment, domain)
				require.True(t, restate.IsTerminalError(err), "expected a terminal ownership error, got %v", err)
				after, err := database.FindFrontlineRouteByFQDN(ctx, domain.domain)
				require.NoError(t, err)
				require.Equal(t, before, after)
			})
		}
	})

	t.Run("registration does not bypass live promotion or rollback", func(t *testing.T) {
		currentDeploymentID := uid.New(uid.DeploymentPrefix)
		for _, rolledBack := range []bool{false, true} {
			require.NoError(t, database.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
				AppID:               app.ID,
				CurrentDeploymentID: sql.NullString{Valid: true, String: currentDeploymentID},
				IsRolledBack:        rolledBack,
				UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			}))
			for _, tt := range []struct {
				sticky       db.FrontlineRoutesSticky
				deploymentID string
			}{
				{db.FrontlineRoutesStickyLive, currentDeploymentID},
				{db.FrontlineRoutesStickyEnvironment, currentDeploymentID},
				{db.FrontlineRoutesStickyBranch, deployment.ID},
				{db.FrontlineRoutesStickyNone, deployment.ID},
				{db.FrontlineRoutesStickyDeployment, deployment.ID},
			} {
				t.Run(string(tt.sticky), func(t *testing.T) {
					domain := newDomain{domain: uid.DNS1035() + ".example.com", sticky: tt.sticky}
					_, err := w.createFrontlineRoute(ctx, deployment, domain)
					require.NoError(t, err)
					route, err := database.FindFrontlineRouteByFQDN(ctx, domain.domain)
					require.NoError(t, err)
					require.Equal(t, tt.deploymentID, route.DeploymentID, "rollback=%t", rolledBack)
				})
			}
		}
	})
}
