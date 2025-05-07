package workflows

import (
	"fmt"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/go/pkg/db"
)

type CountKeysWorkflow struct {
	DB           db.Database
	HeartbeatURL string
}

func (w *CountKeysWorkflow) Run(ctx restate.WorkflowContext) error {

	now := time.Now()

	keyrings, err := restate.Run(ctx, func(ctx restate.RunContext) ([]db.GetOutdatedKeySpacesRow, error) {

		return db.Query.GetOutdatedKeySpaces(ctx, w.DB.RO(), now.UnixMilli())

	})
	if err != nil {
		return err
	}

	ctx.Log().Info("Found keyrings for counting", "keyrings", len(keyrings))

	for _, keyring := range keyrings {

		count, err := restate.Run(ctx, func(ctx restate.RunContext) (int64, error) {
			ctx.Log().Info("counting keys",
				"keyringID", keyring.KeyAuth.ID,
				"workspaceID", keyring.KeyAuth.WorkspaceID,
			)

			return db.Query.CountKeysForKeySpace(ctx, w.DB.RO(), keyring.KeyAuth.ID)
		}, restate.WithName(fmt.Sprintf("counting keys for keyring %s", keyring.KeyAuth.ID)))
		if err != nil {
			return err
		}

		_, err = restate.Run(ctx, func(ctx restate.RunContext) (restate.Void, error) {
			ctx.Log().Info("updating counts",
				"keyringID", keyring.KeyAuth.ID,
				"workspaceID", keyring.KeyAuth.WorkspaceID,
				"count", count,
			)

			err = db.Query.UpdateKeySpaceSize(ctx, w.DB.RW(), db.UpdateKeySpaceSizeParams{
				SizeApprox: int32(count),
				Now:        now.UnixMilli(),
				KeyAuthID:  keyring.KeyAuth.ID,
			})
			return restate.Void{}, err
		}, restate.WithName(fmt.Sprintf("updating key count for keyring %s", keyring.KeyAuth.ID)))
		if err != nil {
			return err
		}

	}

	interval := time.Minute

	nextInvocation := now.Truncate(interval).Add(interval)
	delay := nextInvocation.Sub(time.Now())

	// until restate has cron jobs, we just schedule the workflow to run again in 1min
	restate.WorkflowSend(ctx, "CountKeysWorkflow", fmt.Sprintf("%s", nextInvocation), "Run").
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
