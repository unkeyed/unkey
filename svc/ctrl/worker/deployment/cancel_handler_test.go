package deployment_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestCancelDeploymentAbortsTheRun(t *testing.T) {
	ctx := context.Background()
	f := newHandlerFixture(t, ctx, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED)

	dep := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.projectID,
		AppID:         f.appID,
		EnvironmentID: f.environmentID,
		Status:        mysqltype.DeploymentsStatusBuilding,
	})
	invocationID := uid.New("inv")
	require.NoError(t, f.database.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
		InvocationID: sql.NullString{String: invocationID, Valid: true},
		UpdatedAt:    sql.NullInt64{},
		ID:           dep.ID,
	}))

	actorID := uid.New("user")
	_, err := f.cancel(ctx, dep.ID, actorID)
	require.NoError(t, err)

	f.requireStatus(t, ctx, dep.ID, mysqltype.DeploymentsStatusCancelled)
	require.Equal(t, []string{invocationID}, f.admin.invocations())
	require.Equal(t, 1, f.countAudits(t, ctx, auditlog.DeploymentCancelEvent, dep.ID, actorID))
}

// A deployment waiting for approval has no run yet, so only the row moves
func TestCancelDeploymentAwaitingApproval(t *testing.T) {
	ctx := context.Background()
	f := newHandlerFixture(t, ctx, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED)

	dep := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.projectID,
		AppID:         f.appID,
		EnvironmentID: f.environmentID,
		Status:        mysqltype.DeploymentsStatusAwaitingApproval,
	})

	actorID := uid.New("user")
	_, err := f.cancel(ctx, dep.ID, actorID)
	require.NoError(t, err)

	f.requireStatus(t, ctx, dep.ID, mysqltype.DeploymentsStatusCancelled)
	require.Empty(t, f.admin.invocations())
	require.Equal(t, 1, f.countAudits(t, ctx, auditlog.DeploymentCancelEvent, dep.ID, actorID))
}

func TestCancelDeploymentRejectsFinishedDeployment(t *testing.T) {
	ctx := context.Background()
	f := newHandlerFixture(t, ctx, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED)

	dep := f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.projectID,
		AppID:         f.appID,
		EnvironmentID: f.environmentID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	actorID := uid.New("user")
	_, err := f.cancel(ctx, dep.ID, actorID)
	require.ErrorContains(t, err, "not in progress")

	f.requireStatus(t, ctx, dep.ID, mysqltype.DeploymentsStatusReady)
	require.Empty(t, f.admin.invocations())
	require.Zero(t, f.countAudits(t, ctx, auditlog.DeploymentCancelEvent, dep.ID, actorID))
}

func (f *handlerFixture) cancel(ctx context.Context, deploymentID, actorID string) (*hydrav1.CancelDeploymentResponse, error) {
	return hydrav1.NewDeploymentServiceIngressClient(f.client, deploymentID).
		CancelDeployment().
		Request(ctx, &hydrav1.CancelDeploymentRequest{
			Actor:         &ctrlv1.ActorInfo{Id: actorID, Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
			CorrelationId: uid.New("corr"),
		})
}
