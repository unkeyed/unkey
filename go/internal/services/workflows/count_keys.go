package workflows

import (
	"fmt"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// CountKeysWorkflow manages scheduled jobs that count and update key space sizes.
// It periodically scans for outdated key spaces and updates their approximate sizes
// to provide accurate metrics for monitoring and billing purposes.
type CountKeysWorkflow struct {
	// DB provides database access for counting keys and updating metrics
	DB db.Database
	// HeartbeatURL is an optional URL to ping after successful execution (for monitoring)
	HeartbeatURL string
}

// Run executes the key counting workflow as a restate workflow function.
// It performs the following steps:
// 1. Finds all key spaces that need their counts updated
// 2. Counts the keys in each key space
// 3. Updates the key space size approximation in the database
// 4. Schedules the next run of the workflow
// 5. Sends an optional heartbeat ping for monitoring
//
// This workflow is designed to be executed on a regular schedule and
// maintains its own scheduling by sending a delayed message to itself.
//
// Parameters:
//   - ctx: The restate workflow context
//
// Returns an error if any part of the counting or updating process fails.
func (w *CountKeysWorkflow) Run(wfCtx restate.WorkflowContext) error {

	now := time.Now()

	idCursor := ""
	for {
		keyrings, err := restate.Run(wfCtx, func(ctx restate.RunContext) ([]db.GetOutdatedKeySpacesRow, error) {

			return db.Query.GetOutdatedKeySpaces(ctx, w.DB.RO(), db.GetOutdatedKeySpacesParams{
				IDCursor:   idCursor,
				CutoffTime: now.UnixMilli(),
			})

		})
		if err != nil {
			return err
		}
		if len(keyrings) > 0 {
			idCursor = keyrings[len(keyrings)-1].ID
		} else {
			// break out of the infinite loop
			break
		}

		wfCtx.Log().Info("Found keyrings for counting", "keyrings", len(keyrings))

		promises := []restate.Selectable{}

		for i, keyring := range keyrings {

			promises = append(promises, restate.RunAsync(wfCtx, func(ctx restate.RunContext) (int64, error) {
				ctx.Log().Info("counting keys",
					"keyringID", keyring.ID,
					"batch", i,
				)

				keys, countErr := db.Query.CountKeysForKeySpace(ctx, w.DB.RO(), keyring.ID)
				if countErr != nil {
					return 0, countErr
				}

				err = db.Query.UpdateKeySpaceSize(ctx, w.DB.RW(), db.UpdateKeySpaceSizeParams{
					SizeApprox: int32(keys), // nolint:gosec
					Now:        now.UnixMilli(),
					KeyAuthID:  keyring.ID,
				})
				if err != nil {
					return 0, err
				}
				return keys, nil

			}, restate.WithName(fmt.Sprintf("updating keys for %s", keyring.ID))))

		}

		selector := restate.Select(wfCtx, promises...)
		for selector.Remaining() {
			s, ok := selector.Select().(restate.RunAsyncFuture[int64])
			if !ok {
				return fmt.Errorf("unexpected type")
			}
			_, err = s.Result()
			if err != nil {
				return err
			}
		}

		interval := time.Minute

		nextInvocation := now.Truncate(interval).Add(interval)
		delay := time.Until(nextInvocation)

		// until restate has cron jobs, we just schedule the workflow to run again in 1min
		restate.WorkflowSend(wfCtx, "CountKeysWorkflow", nextInvocation.String(), "Run").
			Send(nil, restate.WithDelay(delay))

		if w.HeartbeatURL != "" {
			_, err = restate.Run(wfCtx, func(ctx restate.RunContext) (restate.Void, error) {
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
	}
	return nil

}
