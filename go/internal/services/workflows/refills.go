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

type RefillWorkflow struct {
	DB           db.Database
	Audit        auditlogs.AuditLogService
	HeartbeatURL string
}

func (w *RefillWorkflow) Run(ctx restate.WorkflowContext) error {

	now := time.Now()

	keys, err := restate.Run(ctx, func(ctx restate.RunContext) ([]db.FindKeysForRefillRow, error) {

		firstOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		lastDayOfMonth := firstOfNextMonth.Add(-24 * time.Hour)
		isLastDayOfMonth := lastDayOfMonth.Day() == now.Day()

		return db.Query.FindKeysForRefill(ctx, w.DB.RO(), db.FindKeysForRefillParams{
			Today:            sql.NullInt16{Valid: true, Int16: int16(now.Day())}, // nolint:gosec
			IsLastDayOfMonth: isLastDayOfMonth,
		})

	})
	if err != nil {
		return err
	}

	ctx.Log().Info("Found keys for refill", "keys", len(keys))

	for _, key := range keys {

		_, err := restate.Run(ctx, func(ctx restate.RunContext) (restate.Void, error) {
			ctx.Log().Info("refilling key",
				"keyID", key.Key.ID,
				"workspaceID", key.Workspace.ID,
				"amount", key.Key.RefillAmount.Int32,
			)
			return restate.Void{}, db.Query.RefillKey(ctx, w.DB.RW(), db.RefillKeyParams{
				KeyID:        key.Key.ID,
				Now:          sql.NullTime{Valid: true, Time: now},
				RefillAmount: sql.NullInt32{Valid: true, Int32: key.Key.RefillAmount.Int32},
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
					WorkspaceID: key.Workspace.ID,
					Display:     fmt.Sprintf("Refilled %s to %d", key.Key.ID, key.Key.RefillAmount.Int32),
					Bucket:      "unkey_mutations",
					ActorType:   "system",
					ActorID:     "system",
					ActorName:   "System",
					ActorMeta:   nil,
					RemoteIP:    "",
					UserAgent:   "",
					Resources: []auditlog.AuditLogResource{
						{
							ID:          key.Key.ID,
							DisplayName: key.Key.Name.String,
							Name:        key.Key.Name.String,
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
		}, restate.WithName(fmt.Sprintf("updating key %s", key.Key.ID)))
		if err != nil {
			return err
		}
	}
	interval := 24 * time.Hour

	nextInvocation := now.Truncate(interval).Add(interval)
	delay := nextInvocation.Sub(time.Now())

	// until restate has cron jobs, we just schedule the workflow to run again in 24h
	restate.WorkflowSend(ctx, "RefillWorkflow", fmt.Sprintf("%s", nextInvocation), "Run").
		Send(nil, restate.WithDelay(delay))

	if w.HeartbeatURL != "" {
		_, err = restate.Run(ctx, func(ctx restate.RunContext) (restate.Void, error) {
			_, err := http.Get(w.HeartbeatURL)
			return restate.Void{}, err
		})
		if err != nil {
			return err
		}
	}
	return nil

}
