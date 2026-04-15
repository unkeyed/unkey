package sentinel

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Resume continues a paused rollout from the next wave after the one that failed.
func (s *RolloutService) Resume(
	ctx restate.ObjectContext,
	_ *hydrav1.SentinelRolloutServiceResumeRequest,
) (*hydrav1.SentinelRolloutServiceResumeResponse, error) {
	state, err := restate.Get[*rolloutState](ctx, stateKeyRollout)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	if state == nil || state.State != statePaused {
		return nil, restate.TerminalError(fmt.Errorf("no paused rollout to resume"))
	}

	state.State = stateInProgress
	state.CurrentWave++
	restate.Set(ctx, stateKeyRollout, state)

	logger.Info("resuming sentinel rollout", "image", state.Image, "wave", state.CurrentWave)
	elapsed := formatDuration(nowMs(ctx) - state.StartedAtMs)
	notifySlack(
		ctx,
		state.SlackWebhookURL,
		"Rollout resumed",
		fmt.Sprintf("Skipping the failed wave, continuing from wave %d/%d. Elapsed since start: %s. Progress: %d/%d, %d failed.",
			state.CurrentWave+1, len(state.Waves), elapsed,
			len(state.SucceededIDs), state.TotalSentinels, len(state.FailedIDs)),
	)

	resp, err := s.executeWaves(ctx, state)
	if err != nil {
		return nil, err
	}

	return &hydrav1.SentinelRolloutServiceResumeResponse{
		State: resp.GetState(),
	}, nil
}
