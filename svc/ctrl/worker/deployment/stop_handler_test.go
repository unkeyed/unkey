package deployment_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestStopProductionDeploymentProtectsLiveDeployment(t *testing.T) {
	ctx := t.Context()
	f := newHandlerFixture(t, ctx, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED)
	environment := f.seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: f.workspaceID,
		ProjectID:   f.projectID,
		AppID:       f.appID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})
	dep := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.projectID,
		AppID:         f.appID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	require.NoError(t, f.database.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		AppID:               f.appID,
		CurrentDeploymentID: sql.NullString{String: dep.ID, Valid: true},
	}))
	client := hydrav1.NewDeploymentServiceIngressClient(f.client, dep.ID)
	request := &hydrav1.StopDeploymentRequest{
		DeploymentId:  dep.ID,
		Actor:         &ctrlv1.ActorInfo{Id: uid.New("user"), Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
		CorrelationId: uid.New("corr"),
	}
	_, err := client.StopDeployment().Request(ctx, request)
	require.Error(t, err)
	live, err := f.database.FindDeploymentById(ctx, dep.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsDesiredStateRunning, live.DesiredState)

	replacement := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.projectID,
		AppID:         f.appID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	require.NoError(t, f.database.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		AppID:               f.appID,
		CurrentDeploymentID: sql.NullString{String: replacement.ID, Valid: true},
	}))
	_, err = client.StopDeployment().Request(ctx, request)
	require.NoError(t, err)
	stopped, err := f.database.FindDeploymentById(ctx, dep.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsDesiredStateStopped, stopped.DesiredState)
	live, err = f.database.FindDeploymentById(ctx, replacement.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsDesiredStateRunning, live.DesiredState)
}
