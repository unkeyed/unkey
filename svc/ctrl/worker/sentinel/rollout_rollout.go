package sentinel

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Rollout starts a progressive rollout of a new sentinel image. It queries all
// running sentinels, splits them into waves, and deploys each wave by fanning
// out SentinelService.Deploy calls. If any sentinel in a wave fails, the
// rollout pauses and returns.
func (s *RolloutService) Rollout(
	ctx restate.ObjectContext,
	req *hydrav1.SentinelRolloutServiceRolloutRequest,
) (*hydrav1.SentinelRolloutServiceRolloutResponse, error) {
	if req.GetImage() == "" {
		return nil, restate.TerminalError(fmt.Errorf("image is required"))
	}

	// Reject if a rollout is already active.
	existing, err := restate.Get[*rolloutState](ctx, stateKeyRollout)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	if existing != nil && existing.State != stateIdle && existing.State != stateCompleted && existing.State != stateCancelled {
		return nil, restate.TerminalError(fmt.Errorf("rollout already active in state %s", existing.State))
	}

	// Collect all running sentinel IDs and validate they're on the same image.
	type sentinelInfo struct {
		ID    string
		Image string
	}

	var allSentinels []sentinelInfo
	afterID := ""
	for {
		rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListRunningSentinelIDsAndImagesRow, error) {
			return db.Query.ListRunningSentinelIDsAndImages(rc, s.db.RO(), db.ListRunningSentinelIDsAndImagesParams{
				AfterID: afterID,
				Limit:   100,
			})
		}, restate.WithName("list running sentinels"))
		if err != nil {
			return nil, fmt.Errorf("list sentinels: %w", err)
		}

		for _, row := range rows {
			allSentinels = append(allSentinels, sentinelInfo{ID: row.ID, Image: row.Image})
			afterID = row.ID
		}

		if len(rows) < 100 {
			break
		}
	}

	if len(allSentinels) == 0 {
		return &hydrav1.SentinelRolloutServiceRolloutResponse{
			State: hydrav1.SentinelRolloutState_SENTINEL_ROLLOUT_STATE_COMPLETED,
		}, nil
	}

	// Build per-sentinel previous image map and filter out sentinels already
	// on the target image (e.g. from a previous partial rollout).
	previousImages := make(map[string]string, len(allSentinels))
	var sentinelIDs []string
	for _, sen := range allSentinels {
		if sen.Image == req.GetImage() {
			continue
		}
		previousImages[sen.ID] = sen.Image
		sentinelIDs = append(sentinelIDs, sen.ID)
	}

	if len(sentinelIDs) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("all sentinels are already on image %s", req.GetImage()))
	}

	wavePercentages := req.GetWavePercentages()
	if len(wavePercentages) == 0 {
		wavePercentages = defaultWavePercentages
	}

	waves := computeWaves(sentinelIDs, wavePercentages)

	slackWebhookURL := req.GetSlackWebhookUrl()
	if slackWebhookURL == "" {
		slackWebhookURL = s.defaultSlackWebhookURL
	}

	state := &rolloutState{
		State:           stateInProgress,
		Image:           req.GetImage(),
		PreviousImages:  previousImages,
		SlackWebhookURL: slackWebhookURL,
		WavePercentages: wavePercentages,
		Waves:           waves,
		CurrentWave:     0,
		SucceededIDs:    []string{},
		FailedIDs:       []string{},
		TotalSentinels:  len(sentinelIDs),
	}
	restate.Set(ctx, stateKeyRollout, state)

	logger.Info("starting sentinel rollout",
		"image", req.GetImage(),
		"sentinels", len(sentinelIDs),
		"waves", len(waves),
	)
	notifySlack(ctx, state.SlackWebhookURL, "Sentinel rollout started",
		fmt.Sprintf("Rolling out `%s` to %d sentinels in %d waves.", req.GetImage(), len(sentinelIDs), len(waves)))

	return s.executeWaves(ctx, state)
}

// executeWaves runs through waves starting from state.CurrentWave. Extracted
// so both Rollout and Resume can call it.
func (s *RolloutService) executeWaves(
	ctx restate.ObjectContext,
	state *rolloutState,
) (*hydrav1.SentinelRolloutServiceRolloutResponse, error) {
	for i := state.CurrentWave; i < len(state.Waves); i++ {
		wave := state.Waves[i]
		state.CurrentWave = i
		restate.Set(ctx, stateKeyRollout, state)

		logger.Info("starting wave", "wave", i, "sentinels", len(wave))
		notifySlack(ctx, state.SlackWebhookURL, "Wave started",
			fmt.Sprintf("Wave %d/%d: deploying to %d sentinels.", i+1, len(state.Waves), len(wave)))

		// Fan out Deploy calls for all sentinels in this wave.
		type deployFuture = restate.ResponseFuture[*hydrav1.SentinelServiceDeployResponse]
		futures := make([]deployFuture, len(wave))
		for j, sentinelID := range wave {
			futures[j] = hydrav1.NewSentinelServiceClient(ctx, sentinelID).
				Deploy().
				RequestFuture(&hydrav1.SentinelServiceDeployRequest{Image: state.Image})
		}

		// Collect results.
		waveFailed := false
		for j, fut := range futures {
			resp, err := fut.Response()
			if err != nil {
				state.FailedIDs = append(state.FailedIDs, wave[j])
				waveFailed = true
				logger.Error("sentinel deploy error", "sentinel_id", wave[j], "error", err)
				continue
			}
			if resp.GetStatus() != hydrav1.SentinelDeployStatus_SENTINEL_DEPLOY_STATUS_READY {
				state.FailedIDs = append(state.FailedIDs, wave[j])
				waveFailed = true
				logger.Warn("sentinel deploy failed", "sentinel_id", wave[j], "status", resp.GetStatus())
			} else {
				state.SucceededIDs = append(state.SucceededIDs, wave[j])
			}
		}

		if waveFailed {
			state.State = statePaused
			restate.Set(ctx, stateKeyRollout, state)
			notifySlack(ctx, state.SlackWebhookURL, "Rollout paused",
				fmt.Sprintf("Wave %d/%d had failures. %d succeeded, %d failed. Investigate and call Resume or RollbackAll.",
					i+1, len(state.Waves), len(state.SucceededIDs), len(state.FailedIDs)))
			return &hydrav1.SentinelRolloutServiceRolloutResponse{
				State: hydrav1.SentinelRolloutState_SENTINEL_ROLLOUT_STATE_PAUSED,
			}, nil
		}

		notifySlack(ctx, state.SlackWebhookURL, "Wave completed",
			fmt.Sprintf("Wave %d/%d complete. %d/%d sentinels updated.", i+1, len(state.Waves), len(state.SucceededIDs), state.TotalSentinels))
	}

	state.State = stateCompleted
	restate.Set(ctx, stateKeyRollout, state)
	notifySlack(ctx, state.SlackWebhookURL, "Rollout completed",
		fmt.Sprintf("All %d sentinels updated to `%s`.", state.TotalSentinels, state.Image))

	logger.Info("sentinel rollout completed", "image", state.Image, "sentinels", state.TotalSentinels)
	return &hydrav1.SentinelRolloutServiceRolloutResponse{
		State: hydrav1.SentinelRolloutState_SENTINEL_ROLLOUT_STATE_COMPLETED,
	}, nil
}
