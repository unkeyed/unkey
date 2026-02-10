package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
)

// how long a deployment must be idle for before we scale it down to 0
var idleTime = 6 * time.Hour

// ScaleDownIdlePreviewDeployments reclaims resources from preview deployments
// that have received no traffic within the idle window defined by idleTime.
// Preview environments can accumulate many running deployments from feature
// branches that are no longer actively used, so this workflow paginates through
// all preview environments and transitions idle deployments to archived by
// checking request counts in ClickHouse.
func (w *Workflow) ScaleDownIdlePreviewDeployments(ctx restate.WorkflowSharedContext, req *hydrav1.ScaleDownIdlePreviewDeploymentsRequest) (*hydrav1.ScaleDownIdlePreviewDeploymentsResponse, error) {

	cutoff := time.Now().Add(-idleTime).UnixMilli()

	cursor := uint64(0)
	for {

		environments, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Environment, error) {
			return db.Query.ListPreviewEnvironments(runCtx, w.db.RO(), db.ListPreviewEnvironmentsParams{
				PaginationCursor: cursor,
				Limit:            100,
			})
		}, restate.WithName("list preview environments"))
		if err != nil {
			return nil, err
		}

		if len(environments) == 0 {
			break
		}
		cursor = environments[len(environments)-1].Pk

		for _, environment := range environments {

			deployments, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Deployment, error) {
				return db.Query.ListDeploymentsByEnvironmentIdAndStatus(runCtx, w.db.RO(), db.ListDeploymentsByEnvironmentIdAndStatusParams{
					EnvironmentID: environment.ID,
					Status:        db.DeploymentsStatusReady,
					CreatedBefore: cutoff,
					UpdatedBefore: sql.NullInt64{Valid: true, Int64: cutoff},
				})
			}, restate.WithName(fmt.Sprintf("get deployments for %s", environment.ID)))
			if err != nil {
				return nil, err
			}

			for _, deployment := range deployments {
				requests, err := restate.Run(ctx, func(runCtx restate.RunContext) (int64, error) {
					return w.clickhouse.GetDeploymentRequestCount(runCtx, clickhouse.GetDeploymentRequestCountRequest{
						WorkspaceID:   deployment.WorkspaceID,
						ProjectID:     deployment.ProjectID,
						EnvironmentID: deployment.EnvironmentID,
						DeploymentID:  deployment.ID,
						Duration:      idleTime,
					})
				}, restate.WithName(fmt.Sprintf("fetch request count for %s", deployment.ID)))
				if err != nil {
					return nil, err
				}

				if requests == 0 {
					_, err = hydrav1.NewDeploymentServiceClient(ctx, deployment.ID).ScheduleDesiredStateChange().Request(&hydrav1.ScheduleDesiredStateChangeRequest{
						DelayMillis: 0,
						State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_ARCHIVED,
					})

					if err != nil {
						return nil, err
					}
				}
			}

		}

	}

	return &hydrav1.ScaleDownIdlePreviewDeploymentsResponse{}, nil
}
