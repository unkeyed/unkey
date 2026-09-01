package environment_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/environment"
)

// TestDeleteCancelsProgressingDeploymentsAndAuditsThem pins the deletion
// contract after the deploycancel consolidation: the cascade drops every
// deployment row, and each PROGRESSING deployment gets a deployment.cancel
// audit entry carrying the deletion's actor, while an already-terminal
// deployment is deleted without one.
func TestDeleteCancelsProgressingDeploymentsAndAuditsThem(t *testing.T) {
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
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "KEBAP",
		Slug:          strings.ToLower(strings.ReplaceAll(uid.New(uid.AppPrefix), "_", "-")),
		DefaultBranch: "main",
	})
	env := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
		Kind:        mysqltype.EnvironmentKindPreview,
	})

	building := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusBuilding,
	})
	finished := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	// A well-formed invocation id that never existed: the admin API answers
	// 404, which CancelInvocation treats as "already finished", so the handler
	// proceeds. A malformed id would draw a 400 instead and retry forever.
	require.NoError(t, database.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
		ID:           building.ID,
		InvocationID: sql.NullString{Valid: true, String: "inv_10szN9fZM3EJ1Y2ejzWDGCIf0AoyEnu4Kz"},
		UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	var svc *environment.Service
	restateCfg := containers.Restate(t, hydrav1.NewEnvironmentServiceServer(&lazyEnvironmentService{svc: &svc}))

	svc, err = environment.New(environment.Config{
		DB:        database,
		Admin:     restateadmin.New(restateadmin.Config{BaseURL: restateCfg.AdminURL, APIKey: ""}),
		Auditlogs: auditlogSvc,
	})
	require.NoError(t, err)

	actorID := uid.New("user")
	_, err = hydrav1.NewEnvironmentServiceIngressClient(restateCfg.IngressClient, env.ID).
		Delete().
		Request(ctx, &hydrav1.DeleteEnvironmentRequest{
			Actor:         &ctrlv1.ActorInfo{Id: actorID, Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
			CorrelationId: uid.New("corr"),
		})
	require.NoError(t, err)

	_, err = database.FindEnvironmentById(ctx, env.ID)
	require.True(t, db.IsNotFound(err), "the environment row must be gone")
	_, err = database.FindDeploymentById(ctx, building.ID)
	require.True(t, db.IsNotFound(err), "the cascade must drop the building deployment")
	_, err = database.FindDeploymentById(ctx, finished.ID)
	require.True(t, db.IsNotFound(err), "the cascade must drop the finished deployment")

	require.Equal(t, 1, countAudits(t, ctx, database, workspaceID, auditlog.DeploymentCancelEvent, building.ID, actorID),
		"a progressing deployment killed by the deletion must be audited with the deletion's actor")
	require.Equal(t, 0, countAudits(t, ctx, database, workspaceID, auditlog.DeploymentCancelEvent, finished.ID, actorID),
		"a deployment that already finished was not cancelled, so it must not be audited as one")
	require.Equal(t, 1, countAudits(t, ctx, database, workspaceID, auditlog.EnvironmentDeleteEvent, env.ID, actorID))
}

// lazyEnvironmentService lets the container bind before the service exists:
// containers.Restate needs the definition up front, while the admin client the
// service depends on needs the container's admin URL.
type lazyEnvironmentService struct {
	hydrav1.UnimplementedEnvironmentServiceServer
	svc **environment.Service
}

func (l *lazyEnvironmentService) Delete(
	ctx restate.ObjectContext,
	req *hydrav1.DeleteEnvironmentRequest,
) (*hydrav1.DeleteEnvironmentResponse, error) {
	return (*l.svc).Delete(ctx, req)
}

func countAudits(t *testing.T, ctx context.Context, database db.Database, workspaceID string, event auditlog.AuditLogEvent, resourceID, actorID string) int {
	t.Helper()
	rows, err := database.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		payload := string(row.Payload)
		if strings.Contains(payload, string(event)) &&
			strings.Contains(payload, resourceID) &&
			strings.Contains(payload, actorID) {
			count++
		}
	}
	return count
}
