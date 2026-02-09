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

func (w *Workflow) ScaleDownIdleDeployments(ctx restate.WorkflowSharedContext, req *hydrav1.ScaleDownIdleDeploymentsRequest) (*hydrav1.ScaleDownIdleDeploymentsResponse, error) {

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
					err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
						return db.Query.UpdateDeploymentDesiredState(runCtx, w.db.RW(), db.UpdateDeploymentDesiredStateParams{
							ID:           deployment.ID,
							UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
							DesiredState: db.DeploymentsDesiredStateStandby,
						})
					}, restate.WithName(fmt.Sprintf("set standby state for %s", deployment.ID)))
					if err != nil {
						return nil, err
					}
				}
			}

		}

	}

	return &hydrav1.ScaleDownIdleDeploymentsResponse{}, nil
}
