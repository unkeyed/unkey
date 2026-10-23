package app_test

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
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/app"
)

// TestDeleteRemovesCustomDomainsWhenEnvironmentDeleteFails pins that deleting
// an app frees its custom domains even when an environment delete never gets
// to them. A leftover row keeps the hostname taken in the workspace, so the
// customer cannot attach it to another app.
func TestDeleteRemovesCustomDomainsWhenEnvironmentDeleteFails(t *testing.T) {
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
	deleted := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "deleted",
		Slug:        slugFrom(uid.New(uid.AppPrefix)),
	})
	kept := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "kept",
		Slug:        slugFrom(uid.New(uid.AppPrefix)),
	})
	deletedEnv := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       deleted.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})
	keptEnv := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       kept.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})

	deletedDomain := insertVerifiedDomain(t, database, project.ID, deleted.ID, deletedEnv.ID, workspaceID, uid.DNS1035(12)+".example.com")
	keptDomain := insertVerifiedDomain(t, database, project.ID, kept.ID, keptEnv.ID, workspaceID, uid.DNS1035(12)+".example.com")

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	appSvc, err := app.New(app.Config{DB: database, Auditlogs: auditlogSvc})
	require.NoError(t, err)

	// Every environment delete fails terminally, so only the app delete can
	// remove the domain.
	restateCfg := containers.Restate(t,
		hydrav1.NewAppServiceServer(appSvc),
		hydrav1.NewEnvironmentServiceServer(hydrav1.UnimplementedEnvironmentServiceServer{}),
	)

	_, err = hydrav1.NewAppServiceIngressClient(restateCfg.IngressClient, deleted.ID).
		Delete().
		Request(ctx, &hydrav1.DeleteAppRequest{
			Actor:         &ctrlv1.ActorInfo{Id: uid.New("user"), Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
			CorrelationId: uid.New("corr"),
		})
	require.NoError(t, err)

	_, err = database.FindCustomDomainById(ctx, deletedDomain)
	require.True(t, db.IsNotFound(err), "the deleted app's custom domain must be gone")
	routes, err := database.CountFrontlineRoutesByAppId(ctx, deleted.ID)
	require.NoError(t, err)
	require.Zero(t, routes, "the deleted app's frontline routes must be gone")

	_, err = database.FindCustomDomainById(ctx, keptDomain)
	require.NoError(t, err, "another app's custom domain must survive")
	routes, err = database.CountFrontlineRoutesByAppId(ctx, kept.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), routes, "another app's frontline route must survive")
}

func insertVerifiedDomain(t *testing.T, database db.Database, projectID, appID, environmentID, workspaceID, domain string) string {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UnixMilli()

	domainID := uid.New(uid.DomainPrefix)
	require.NoError(t, database.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
		ID:                    domainID,
		WorkspaceID:           workspaceID,
		ProjectID:             projectID,
		AppID:                 appID,
		EnvironmentID:         environmentID,
		Domain:                domain,
		ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
		VerificationStatus:    db.CustomDomainsVerificationStatusVerified,
		VerificationToken:     uid.Secure(24),
		TargetCname:           uid.DNS1035(16) + ".unkey-dns.com",
		DomainConnectProvider: sql.NullString{},
		DomainConnectUrl:      sql.NullString{},
		InvocationID:          sql.NullString{},
		CreatedAt:             now,
	}))
	require.NoError(t, database.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                projectID,
		AppID:                    appID,
		DeploymentID:             "",
		EnvironmentID:            environmentID,
		FullyQualifiedDomainName: domain,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Valid: true, Int64: now},
	}))
	return domainID
}

func slugFrom(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "_", "-"))
}
