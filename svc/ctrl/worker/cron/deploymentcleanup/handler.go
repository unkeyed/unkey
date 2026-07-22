// Package deploymentcleanup implements the CronService.RunDeploymentCleanup
// handler. The handler prunes deployment rows and external Depot resources.
// It first hard-deletes deployments that reached a
// non-recoverable terminal status (failed, cancelled, superseded, skipped)
// more than the retention window ago. Image tags are reconciled separately:
// a deployment row does not prove exclusive ownership because user-supplied
// images and rebuilds can share the same tag.
package deploymentcleanup

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/registrysweep"
)

// retention is how long a non-recoverable deployment is kept before this
// sweep deletes it. The window preserves recent history in the dashboard.
const retention = 30 * 24 * time.Hour

// batchLimit bounds each list/delete round so row locks stay short.
const batchLimit int32 = 100

// maxBatchesPerRun caps a single invocation so its Restate journal stays
// bounded. A larger backlog drains across daily runs.
const maxBatchesPerRun = 50

var prunableDeploymentStatuses = []mysqltype.DeploymentsStatus{
	mysqltype.DeploymentsStatusFailed,
	mysqltype.DeploymentsStatusCancelled,
	mysqltype.DeploymentsStatusSuperseded,
	mysqltype.DeploymentsStatusSkipped,
}

// Config holds the handler's dependencies.
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database

	// Heartbeat is pinged after a successful sweep. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat

	// RegistrySweep reconciles external deployment resources. Must not be nil.
	RegistrySweep *registrysweep.Handler
}

// Handler executes RunDeploymentCleanup.
type Handler struct {
	db            db.Database
	heartbeat     healthcheck.Heartbeat
	registrySweep *registrysweep.Handler
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.RegistrySweep, "RegistrySweep must not be nil"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, heartbeat: cfg.Heartbeat, registrySweep: cfg.RegistrySweep}, nil
}

// Handle deletes prunable deployments in bounded batches, then reconciles
// external deployment resources. Each database batch is revalidated and
// deleted in one MySQL transaction so a status or live-pointer change between
// listing and deletion cannot remove a newly active deployment.
//
// The VO key is fixed at "deployment-cleanup" and owns the external
// reconciliation phase's pagination state.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunDeploymentCleanupRequest,
) (*hydrav1.RunDeploymentCleanupResponse, error) {
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get now: %w", err)
	}
	cutoff := now.Add(-retention).UnixMilli()

	var deploymentsDeleted int64
	for batchNum := 0; ; batchNum++ {
		if batchNum >= maxBatchesPerRun {
			logger.Info("deployment cleanup reached the per-run batch cap, leaving the rest for the next run",
				"deployments_deleted", deploymentsDeleted,
			)
			break
		}

		ids, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
			return h.db.ListPrunableDeployments(rc, db.ListPrunableDeploymentsParams{
				Statuses: prunableDeploymentStatuses,
				Cutoff:   sql.NullInt64{Int64: cutoff, Valid: true},
				Limit:    batchLimit,
			})
		}, restate.WithName(fmt.Sprintf("list batch-%d", batchNum)), restate.WithMaxRetryAttempts(5))
		if err != nil {
			return nil, fmt.Errorf("list prunable deployments batch %d: %w", batchNum, err)
		}
		if len(ids) == 0 {
			break
		}

		deleted, err := h.deleteRows(ctx, batchNum, ids, cutoff)
		if err != nil {
			return nil, err
		}
		deploymentsDeleted += deleted

		if len(ids) < int(batchLimit) || deleted == 0 {
			break
		}
	}

	registryResult, err := h.registrySweep.Sweep(ctx)
	if err != nil {
		return nil, fmt.Errorf("reconcile deployment resources: %w", err)
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat"), restate.WithMaxRetryAttempts(5)); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}
	logger.Info("deployment resource pruning completed",
		"deployments_deleted", deploymentsDeleted,
		"tags_deleted", registryResult.TagsDeleted,
		"tags_skipped", registryResult.TagsSkipped,
		"depot_projects_deleted", registryResult.DepotProjectsDeleted,
	)

	return &hydrav1.RunDeploymentCleanupResponse{
		DeploymentsDeleted:   deploymentsDeleted,
		TagsDeleted:          registryResult.TagsDeleted,
		TagsSkipped:          registryResult.TagsSkipped,
		DepotProjectsDeleted: registryResult.DepotProjectsDeleted,
	}, nil
}

// deleteRows revalidates and removes one batch atomically.
func (h *Handler) deleteRows(ctx restate.ObjectContext, batchNum int, ids []string, cutoff int64) (int64, error) {
	deleted, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		tx, beginErr := h.db.RW().Begin(rc)
		if beginErr != nil {
			return 0, fmt.Errorf("begin transaction: %w", beginErr)
		}
		committed := false
		defer func() {
			if !committed {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					logger.Error("rollback deployment cleanup transaction", "error", rollbackErr)
				}
			}
		}()

		queries := db.NewQueries(tx)
		eligible, filterErr := queries.FilterPrunableDeploymentIds(rc, db.FilterPrunableDeploymentIdsParams{
			Ids:      ids,
			Statuses: prunableDeploymentStatuses,
			Cutoff:   sql.NullInt64{Int64: cutoff, Valid: true},
		})
		if filterErr != nil {
			return 0, fmt.Errorf("revalidate deployments: %w", filterErr)
		}
		if len(eligible) == 0 {
			return 0, nil
		}

		if deleteErr := deleteDeploymentRows(rc, queries, eligible); deleteErr != nil {
			return 0, deleteErr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return 0, fmt.Errorf("commit transaction: %w", commitErr)
		}
		committed = true
		return int64(len(eligible)), nil
	}, restate.WithName(fmt.Sprintf("delete batch-%d", batchNum)), restate.WithMaxRetryAttempts(5))
	if err != nil {
		return 0, fmt.Errorf("delete batch %d: %w", batchNum, err)
	}
	return deleted, nil
}

// deleteDeploymentRows removes dependent rows before their deployments.
func deleteDeploymentRows(ctx restate.RunContext, queries *db.Queries, ids []string) error {
	if err := queries.DeleteDeploymentStepsByDeploymentIds(ctx, ids); err != nil {
		return fmt.Errorf("delete steps: %w", err)
	}
	if err := queries.DeleteDeploymentTopologiesByDeploymentIds(ctx, ids); err != nil {
		return fmt.Errorf("delete topologies: %w", err)
	}
	if err := queries.DeleteInstancesByDeploymentIds(ctx, ids); err != nil {
		return fmt.Errorf("delete instances: %w", err)
	}
	if err := queries.DeleteFrontlineRoutesByDeploymentIds(ctx, ids); err != nil {
		return fmt.Errorf("delete frontline routes: %w", err)
	}
	if err := queries.DeleteCiliumNetworkPoliciesByDeploymentIds(ctx, ids); err != nil {
		return fmt.Errorf("delete network policies: %w", err)
	}
	nullableIDs := make([]sql.NullString, len(ids))
	for i, id := range ids {
		nullableIDs[i] = sql.NullString{String: id, Valid: true}
	}
	if err := queries.DeleteOpenapiSpecsByDeploymentIds(ctx, nullableIDs); err != nil {
		return fmt.Errorf("delete openapi specs: %w", err)
	}
	if err := queries.DeleteDeploymentsByIds(ctx, ids); err != nil {
		return fmt.Errorf("delete deployments: %w", err)
	}
	return nil
}
