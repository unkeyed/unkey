package sentinel

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// RollbackAll reverts all successfully updated sentinels to the previous image.
// Each sentinel is rolled back via SentinelService.Deploy, which handles
// per-sentinel convergence.
func (s *RolloutService) RollbackAll(
	ctx restate.ObjectContext,
	_ *hydrav1.SentinelRolloutServiceRollbackAllRequest,
) (*hydrav1.SentinelRolloutServiceRollbackAllResponse, error) {
	state, err := restate.Get[*rolloutState](ctx, stateKeyRollout)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	if state == nil {
		return nil, restate.TerminalError(fmt.Errorf("no active rollout"))
	}

	if state.State != statePaused && state.State != stateCancelled {
		return nil, restate.TerminalError(fmt.Errorf("cannot rollback in state %s, must be paused or cancelled", state.State))
	}

	if len(state.SucceededIDs) == 0 {
		state.State = stateCancelled
		restate.Set(ctx, stateKeyRollout, state)
		return &hydrav1.SentinelRolloutServiceRollbackAllResponse{Reverted: 0}, nil
	}

	state.State = stateRollingBack
	restate.Set(ctx, stateKeyRollout, state)

	logger.Info("rolling back all sentinels", "count", len(state.SucceededIDs))
	rollbackStartedAt := nowMs(ctx)
	notifySlack(
		ctx,
		state.SlackWebhookURL,
		"Rollback started",
		fmt.Sprintf("Reverting *%d sentinels* to their previous images. Rollout `%s` ran for %s before rollback.",
			len(state.SucceededIDs), state.Image, formatDuration(rollbackStartedAt-state.StartedAtMs)),
	)

	// Fan out Deploy calls to revert each sentinel to its previous image.
	type deployFuture = restate.ResponseFuture[*hydrav1.SentinelServiceDeployResponse]
	futures := make([]deployFuture, len(state.SucceededIDs))
	for i, sentinelID := range state.SucceededIDs {
		futures[i] = hydrav1.NewSentinelServiceClient(ctx, sentinelID).
			Deploy().
			RequestFuture(&hydrav1.SentinelServiceDeployRequest{Image: state.PreviousImages[sentinelID]})
	}

	reverted := int32(0)
	for i, fut := range futures {
		resp, err := fut.Response()
		if err != nil {
			logger.Error("rollback failed for sentinel", "sentinel_id", state.SucceededIDs[i], "error", err)
			continue
		}

		if resp.GetStatus() == hydrav1.SentinelDeployStatus_SENTINEL_DEPLOY_STATUS_READY {
			reverted++
		} else {
			logger.Warn("rollback did not converge for sentinel", "sentinel_id", state.SucceededIDs[i])
		}
	}

	state.State = stateCancelled
	restate.Set(ctx, stateKeyRollout, state)

	rollbackDuration := formatDuration(nowMs(ctx) - rollbackStartedAt)
	logger.Info("rollback completed", "reverted", reverted, "total", len(state.SucceededIDs), "duration", rollbackDuration)
	notifySlack(ctx, state.SlackWebhookURL, "Rollback completed",
		fmt.Sprintf("Reverted *%d/%d* sentinels to their previous images in %s.",
			reverted, len(state.SucceededIDs), rollbackDuration))

	return &hydrav1.SentinelRolloutServiceRollbackAllResponse{Reverted: reverted}, nil
}
