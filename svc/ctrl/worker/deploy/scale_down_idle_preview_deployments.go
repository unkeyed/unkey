package deploy

import (
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// how long a deployment must be idle for before we scale it down to 0
var idleTime = 6 * time.Hour

// ScaleDownIdlePreviewDeployments reclaims resources from preview deployments
// that have received no traffic within the idle window defined by idleTime.
// Preview environments can accumulate many running deployments from feature
// branches that are no longer actively used, so this workflow paginates through
// all preview environments and transitions idle deployments to archived by
// checking request counts in ClickHouse.
//
// Superseded and unreachable: cron/idlepreview runs the live scan on a 1h idle
// window, and DeployService's router binds no ScaleDownIdlePreviewDeployments
// handler, so nothing can invoke this. It stays as the reference for the
// paginate-environments-then-check-ClickHouse shape.
func (w *Workflow) ScaleDownIdlePreviewDeployments(ctx restate.ObjectContext, req *hydrav1.RunScaleDownIdlePreviewDeploymentsRequest) (*hydrav1.RunScaleDownIdlePreviewDeploymentsResponse, error) {
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, err
	}
	cutoff := now.Add(-idleTime).UnixMilli()

	cursor := uint64(0)
	for {

		environments, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Environment, error) {
			return w.db.ListPreviewEnvironments(runCtx, db.ListPreviewEnvironmentsParams{
				PaginationCursor: cursor,
				Limit:            100,
			})
		}, restate.WithName("list preview environments"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return nil, err
		}

		if len(environments) == 0 {
			break
		}
		cursor = environments[len(environments)-1].Pk

		for _, environment := range environments {

			deployments, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Deployment, error) {
				return w.db.ListDeploymentsByEnvironmentIdAndStatus(runCtx, db.ListDeploymentsByEnvironmentIdAndStatusParams{
					EnvironmentID: environment.ID,
					Status:        mysqltype.DeploymentsStatusReady,
					CreatedBefore: cutoff,
					UpdatedBefore: sql.NullInt64{Valid: true, Int64: cutoff},
				})
			}, restate.WithName(fmt.Sprintf("get deployments for %s", environment.ID)), restate.WithMaxRetryAttempts(runMaxAttempts))
			if err != nil {
				return nil, err
			}

			for _, deployment := range deployments {
				requests, err := restate.Run(ctx, func(runCtx restate.RunContext) (int64, error) {
					return w.clickhouse.GetDeploymentRequestCount(runCtx, clickhouse.GetDeploymentRequestCountRequest{
						WorkspaceID:   deployment.WorkspaceID,
						ProjectID:     deployment.ProjectID,
						AppID:         deployment.AppID,
						EnvironmentID: deployment.EnvironmentID,
						DeploymentID:  deployment.ID,
						Duration:      idleTime,
					})
				}, restate.WithName(fmt.Sprintf("fetch request count for %s", deployment.ID)), restate.WithMaxRetryAttempts(runMaxAttempts))
				if err != nil {
					return nil, err
				}

				if requests == 0 {
					_, err = hydrav1.NewDeploymentServiceClient(ctx, deployment.ID).ScheduleDesiredStateChange().Request(&hydrav1.ScheduleDesiredStateChangeRequest{
						DelayMillis: 0,
						State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
					})
					if err != nil {
						return nil, err
					}
				}
			}

		}

	}

	return &hydrav1.RunScaleDownIdlePreviewDeploymentsResponse{}, nil
}
