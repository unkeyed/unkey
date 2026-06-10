// Package permanentdelete implements the CronService.RunPermanentDelete
// handler. The handler reads the deletions table (the source of truth
// for soft-deleted resources) and dispatches each due row to the
// resource-type's hard-delete VO via a synchronous Request, then
// removes the deletions row.
//
// The cron handler owns deletions-row cleanup: after a successful
// dispatch, the row is removed here. This keeps the chain uniform
// regardless of the cascade root — an app-only delete and a
// project-rooted cascade clean up via the same path.
//
// Adding a new soft-deletable resource type means: handle its
// resource_type string in [Handler.dispatch] and write a
// MarkForDeletion/Restore path on its VO. The cron handler itself
// doesn't change.
package permanentdelete

import (
	"errors"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Resource type values. Must match the strings written to the
// deletions table by MarkForDeletion handlers (worker/{project,app,
// environment}/mark_for_deletion_handler.go).
const (
	resourceTypeProject     = "project"
	resourceTypeApp         = "app"
	resourceTypeEnvironment = "environment"
)

// batchLimit bounds how many rows a single tick processes. The hard
// delete cascade for a project can fan out to many apps/envs/etc, so
// the cap protects Restate from a flood after a long outage.
const batchLimit = 100

// Config holds the handler's dependencies.
type Config struct {
	// DB is the application database. Must not be nil.
	DB db.Database
	// Clock provides the cutoff timestamp. Must not be nil.
	Clock clock.Clock
}

// Handler executes RunPermanentDelete.
type Handler struct {
	db    db.Database
	clock clock.Clock
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clock, "Clock must not be nil"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, clock: cfg.Clock}, nil
}

// Handle selects every deletion whose grace window has elapsed,
// dispatches each row to the resource-type's hard-delete path, and
// removes the deletions row after a successful dispatch. Failures are
// aggregated; a single bad row doesn't poison the batch and leaves
// its deletions row intact for the next tick to retry.
//
// Stateless — the VO key is fixed at "permanent-delete" so a
// paused/wedged invocation cannot block other cron handlers.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunPermanentDeleteRequest,
) (*hydrav1.RunPermanentDeleteResponse, error) {
	cutoff := h.clock.Now().UnixMilli()

	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.Deletion, error) {
		return db.Query.ListDueDeletions(rc, h.db.RO(), db.ListDueDeletionsParams{
			DeletePermanentlyAt: cutoff,
			Limit:               batchLimit,
		})
	}, restate.WithName("list due deletions"))
	if err != nil {
		return nil, fmt.Errorf("list due deletions: %w", err)
	}

	results := map[string]int32{}
	var aggregate error
	var total int32

	for _, row := range rows {
		if err := h.dispatch(ctx, row); err != nil {
			aggregate = errors.Join(aggregate, fmt.Errorf("dispatch %s/%s: %w", row.ResourceType, row.ResourceID, err))
			continue
		}

		// Cascade succeeded. Remove the deletions row by id so the next
		// tick stops dispatching this cascade. Wrap in restate.RunVoid
		// for journaled retry on transient DB errors.
		if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
			return db.Query.DeleteDeletionById(rc, h.db.RW(), row.ID)
		}, restate.WithName("delete deletion row")); err != nil {
			aggregate = errors.Join(aggregate, fmt.Errorf("delete deletion row %s: %w", row.ID, err))
			continue
		}

		results[row.ResourceType]++
		total++
	}

	sweeps := make([]*hydrav1.PermanentDeleteSweepResult, 0, len(results))
	for resource, triggered := range results {
		sweeps = append(sweeps, &hydrav1.PermanentDeleteSweepResult{
			Resource:  resource,
			Triggered: triggered,
		})
	}

	logger.Info("permanent delete sweep complete",
		"total_triggered", total,
		"cutoff_ms", cutoff,
		"failed", aggregate != nil,
	)

	resp := &hydrav1.RunPermanentDeleteResponse{
		TotalTriggered: total,
		Sweeps:         sweeps,
	}
	return resp, aggregate
}

// dispatch routes one deletion row to the right hard-delete VO. The
// VOs are invoked synchronously via Request so the caller knows when
// the cascade has finished and can remove the deletions row.
func (h *Handler) dispatch(ctx restate.ObjectContext, row db.Deletion) error {
	switch row.ResourceType {
	case resourceTypeProject:
		_, err := hydrav1.NewProjectServiceClient(ctx, row.ResourceID).
			DeletePermanently().
			Request(&hydrav1.DeleteProjectPermanentlyRequest{})
		return err
	case resourceTypeApp:
		_, err := hydrav1.NewAppServiceClient(ctx, row.ResourceID).
			DeletePermanently().
			Request(&hydrav1.DeleteAppPermanentlyRequest{})
		return err
	case resourceTypeEnvironment:
		_, err := hydrav1.NewEnvironmentServiceClient(ctx, row.ResourceID).
			DeletePermanently().
			Request(&hydrav1.DeleteEnvironmentPermanentlyRequest{})
		return err
	default:
		return fmt.Errorf("unknown resource_type %q", row.ResourceType)
	}
}
