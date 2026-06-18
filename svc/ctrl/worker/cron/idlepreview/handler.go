// Package idlepreview implements the
// CronService.RunScaleDownIdlePreviewDeployments handler.
package idlepreview

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
)

const (
	// idleTime is the minimum quiet period before a preview deployment can stop.
	idleTime = time.Hour

	// runMaxAttempts bounds retries for each journaled Restate side effect.
	runMaxAttempts = 5
)

// Config holds the dependencies for idle preview deployment cleanup.
type Config struct {
	// DB stores deployment and environment state.
	DB db.Database

	// Clickhouse stores request telemetry used to decide whether a deployment is idle.
	Clickhouse clickhouse.ClickHouse
}

// Handler owns the idle preview deployment scan and stop transition scheduling.
type Handler struct {
	db         db.Database
	clickhouse clickhouse.ClickHouse
}

// New constructs a Handler with the dependencies required to evaluate preview
// deployment idleness.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil; use clickhouse.NewNoop() if unavailable"),
	); err != nil {
		return nil, err
	}
	return &Handler{
		db:         cfg.DB,
		clickhouse: cfg.Clickhouse,
	}, nil
}

// Handle reclaims resources from preview deployments that have received no
// traffic within the idle window. Preview environments can accumulate many
// running deployments from feature branches that are no longer actively used,
// so this workflow paginates through all preview environments and transitions
// idle deployments to stopped.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunScaleDownIdlePreviewDeploymentsRequest,
) (*hydrav1.RunScaleDownIdlePreviewDeploymentsResponse, error) {
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, err
	}
	cutoff := now.Add(-idleTime).UnixMilli()

	cursor := uint64(0)
	for {
		environments, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Environment, error) {
			return db.Query.ListPreviewEnvironments(runCtx, h.db.RO(), db.ListPreviewEnvironmentsParams{
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
				return db.Query.ListDeploymentsByEnvironmentIdAndStatus(runCtx, h.db.RO(), db.ListDeploymentsByEnvironmentIdAndStatusParams{
					EnvironmentID: environment.ID,
					Status:        db.DeploymentsStatusReady,
					CreatedBefore: cutoff,
					UpdatedBefore: sql.NullInt64{Valid: true, Int64: cutoff},
				})
			}, restate.WithName(fmt.Sprintf("get deployments for %s", environment.ID)), restate.WithMaxRetryAttempts(runMaxAttempts))
			if err != nil {
				return nil, err
			}

			for _, deployment := range deployments {
				requests, err := restate.Run(ctx, func(runCtx restate.RunContext) (int64, error) {
					return h.clickhouse.GetDeploymentRequestCount(runCtx, clickhouse.GetDeploymentRequestCountRequest{
						WorkspaceID:   deployment.WorkspaceID,
						ProjectID:     deployment.ProjectID,
						EnvironmentID: deployment.EnvironmentID,
						DeploymentID:  deployment.ID,
						Duration:      idleTime,
					})
				}, restate.WithName(fmt.Sprintf("fetch request count for %s", deployment.ID)), restate.WithMaxRetryAttempts(runMaxAttempts))
				if err != nil {
					return nil, err
				}

				if requests == 0 {
					_, err = hydrav1.NewDeploymentServiceClient(ctx, deployment.ID).
						ScheduleDesiredStateChange().
						Request(&hydrav1.ScheduleDesiredStateChangeRequest{
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
