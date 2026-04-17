package sentinel

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Cancel stops a rollout. Successfully updated sentinels stay on the new image.
// Failed sentinels remain on whatever state they ended up in — use RollbackAll
// to explicitly revert succeeded sentinels to the previous image.
func (s *RolloutService) Cancel(
	ctx restate.ObjectContext,
	_ *hydrav1.SentinelRolloutServiceCancelRequest,
) (*hydrav1.SentinelRolloutServiceCancelResponse, error) {
	state, err := restate.Get[*rolloutState](ctx, stateKeyRollout)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	if state == nil {
		return nil, restate.TerminalError(fmt.Errorf("no active rollout"))
	}

	if state.State != stateInProgress && state.State != statePaused {
		return nil, restate.TerminalError(fmt.Errorf("cannot cancel rollout in state %s", state.State))
	}

	state.State = stateCancelled
	restate.Set(ctx, stateKeyRollout, state)

	logger.Info("sentinel rollout cancelled", "image", state.Image,
		"succeeded", len(state.SucceededIDs), "failed", len(state.FailedIDs))

	notifySlack(
		ctx,
		state.SlackWebhookURL,
		"Rollout cancelled",
		fmt.Sprintf("Rollout of `%s` cancelled. %d sentinels on new image, %d failed.",
			state.Image,
			len(state.SucceededIDs),
			len(state.FailedIDs),
		),
	)

	return &hydrav1.SentinelRolloutServiceCancelResponse{}, nil
}
