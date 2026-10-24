package customdomain

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// A route step can commit its insert and still fail before Restate journals
// the result, so the retry must find the route it already wrote instead of
// failing on the unique FQDN for the rest of the invocation. Another app's route
// for the same FQDN must stay untouched and be reported as not routed.
func TestEnsureFrontlineRouteIsIdempotent(t *testing.T) {
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
		Name:        "routes",
		Slug:        slugFrom(uid.New(uid.ProjectPrefix)),
	})
	newDomain := func() db.CustomDomain {
		app := seeder.CreateApp(ctx, seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: workspaceID,
			ProjectID:   project.ID,
			Name:        "app",
			Slug:        slugFrom(uid.New(uid.AppPrefix)),
		})
		env := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspaceID,
			ProjectID:   project.ID,
			AppID:       app.ID,
			Slug:        "production",
			Kind:        mysqltype.EnvironmentKindProduction,
		})
		return db.CustomDomain{
			ID:            uid.New(uid.DomainPrefix),
			WorkspaceID:   workspaceID,
			ProjectID:     project.ID,
			AppID:         app.ID,
			EnvironmentID: env.ID,
		}
	}

	fqdn := uid.DNS1035(12) + ".example.com"
	owner := newDomain()
	owner.Domain = fqdn
	other := newDomain()
	other.Domain = fqdn

	svc := New(Config{DB: database, CnameDomain: "cname.unkey.local"})
	now := time.Now().UnixMilli()

	routed, err := svc.ensureFrontlineRoute(ctx, owner, now)
	require.NoError(t, err)
	require.True(t, routed, "the first attempt must create the route")

	routed, err = svc.ensureFrontlineRoute(ctx, owner, now)
	require.NoError(t, err, "a retry after the insert committed must not fail on the duplicate FQDN")
	require.True(t, routed, "the route from the earlier attempt must count as done")

	routed, err = svc.ensureFrontlineRoute(ctx, other, now)
	require.NoError(t, err)
	require.False(t, routed, "a route owned by another app must not be reported as routed")

	route, err := database.FindFrontlineRouteByFQDN(ctx, fqdn)
	require.NoError(t, err)
	require.Equal(t, owner.AppID, route.AppID, "another app must not take over the route")
	require.Equal(t, owner.EnvironmentID, route.EnvironmentID)
}

func slugFrom(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "_", "-"))
}
