package workflows

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// RefillWorkflow manages scheduled jobs to refill API keys based on their configured refill schedules.
// It handles periodic refills of keys that have a refill amount and schedule configured,
// refreshing their remaining usage quota according to their settings.
type RefillWorkflow struct {
	// DB provides database access for finding keys and updating their remaining usage
	DB db.Database
	// Audit is the service used to record audit logs of key refill operations
	Audit auditlogs.AuditLogService
	// HeartbeatURL is an optional URL to ping after successful execution (for monitoring)
	HeartbeatURL string
}

// Run executes the key refill workflow as a restate workflow function.
// It performs the following steps:
// 1. Finds all keys that need to be refilled based on their schedules
// 2. Updates each key's remaining usage with the configured refill amount
// 3. Records an audit log for each refill operation
// 4. Schedules the next run of the workflow
// 5. Sends an optional heartbeat ping for monitoring
//
// This workflow supports both day-of-month based refills (e.g., "1st of every month")
// and last-day-of-month refills. It's designed to be executed daily and will only
// refill keys that are scheduled for the current day.
//
// Parameters:
//   - ctx: The restate workflow context
//
// Returns an error if any part of the refill process fails.
func (w *RefillWorkflow) Run(ctx restate.WorkflowContext) error {

	now := time.Now()

	keys, err := restate.Run(ctx, func(ctx restate.RunContext) ([]db.Key, error) {

		firstOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		lastDayOfMonth := firstOfNextMonth.Add(-24 * time.Hour)
		isLastDayOfMonth := lastDayOfMonth.Day() == now.Day()

		return db.Query.FindKeysForRefill(ctx, w.DB.RO(), db.FindKeysForRefillParams{
			Today:            sql.NullInt16{Valid: true, Int16: int16(now.Day())}, // nolint:gosec
			IsLastDayOfMonth: isLastDayOfMonth,
			Cutoff:           sql.NullTime{Valid: true, Time: now.Add(-23*time.Hour - 50*time.Minute)},
		})

	})
	if err != nil {
		return err
	}

	ctx.Log().Info("Found keys for refill", "keys", len(keys))

	for _, key := range keys {

		_, err = restate.Run(ctx, func(ctx restate.RunContext) (restate.Void, error) {
			ctx.Log().Info("refilling key",
				"keyID", key.ID,
				"workspaceID", key.WorkspaceID,
				"amount", key.RefillAmount.Int32,
			)
			return restate.Void{}, db.Query.RefillKey(ctx, w.DB.RW(), db.RefillKeyParams{
				KeyID:        key.ID,
				Now:          sql.NullTime{Valid: true, Time: now},
				RefillAmount: sql.NullInt32{Valid: true, Int32: key.RefillAmount.Int32},
			})
		}, restate.WithName("fetching keys"))
		if err != nil {
			return err
		}

		_, err = restate.Run(ctx, func(ctx restate.RunContext) (restate.Void, error) {

			tx, txErr := w.DB.RW().Begin(ctx)
			if txErr != nil {
				return restate.Void{}, txErr
			}
			defer func() {
				rollbackErr := tx.Rollback()
				if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
					ctx.Log().Error("rollback failed",
						"error", rollbackErr,
					)
				}
			}()

			err = w.Audit.Insert(ctx, tx, []auditlog.AuditLog{
				{
					Event:       "key.update",
					WorkspaceID: key.WorkspaceID,
					Display:     fmt.Sprintf("Refilled %s to %d", key.ID, key.RefillAmount.Int32),
					Bucket:      "unkey_mutations",
					ActorType:   "system",
					ActorID:     "system",
					ActorName:   "System",
					ActorMeta:   nil,
					RemoteIP:    "",
					UserAgent:   "",
					Resources: []auditlog.AuditLogResource{
						{
							ID:          key.ID,
							DisplayName: key.Name.String,
							Name:        key.Name.String,
							Meta:        nil,
							Type:        auditlog.KeyResourceType,
						},
					},
				},
			})

			if err != nil {
				return restate.Void{}, err
			}

			err = tx.Commit()
			if err != nil {
				return restate.Void{}, err
			}

			return restate.Void{}, nil
		}, restate.WithName(fmt.Sprintf("updating key %s", key.ID)))
		if err != nil {
			return err
		}
	}
	interval := 24 * time.Hour

	nextInvocation := now.Truncate(interval).Add(interval)
	delay := time.Until(nextInvocation)

	// until restate has cron jobs, we just schedule the workflow to run again in 24h
	restate.WorkflowSend(ctx, "RefillWorkflow", nextInvocation.String(), "Run").
		Send(nil, restate.WithDelay(delay))

	if w.HeartbeatURL != "" {
		_, err = restate.Run(ctx, func(ctx restate.RunContext) (restate.Void, error) {
			res, heartbeatErr := http.Get(w.HeartbeatURL)
			if heartbeatErr != nil {
				return restate.Void{}, heartbeatErr
			}
			return restate.Void{}, res.Body.Close()
		})
		if err != nil {
			return err
		}
	}
	return nil

}
