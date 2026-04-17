package sentinel

import "math"

const stateKeyRollout = "rollout"

// rolloutState is persisted in Restate K/V state under [stateKeyRollout].
// It tracks the full lifecycle of a rollout so handlers can resume, cancel,
// or rollback without losing progress.
type rolloutState struct {
	State           string            `json:"state"`
	Image           string            `json:"image"`
	PreviousImages  map[string]string `json:"previousImages"`
	SlackWebhookURL string            `json:"slackWebhookUrl,omitempty"`
	WavePercentages []int32           `json:"wavePercentages"`
	Waves           [][]string        `json:"waves"`
	CurrentWave     int               `json:"currentWave"`
	SucceededIDs    []string          `json:"succeededIds"`
	FailedIDs       []string          `json:"failedIds"`
	TotalSentinels  int               `json:"totalSentinels"`
}

const (
	stateIdle        = "idle"
	stateInProgress  = "in_progress"
	statePaused      = "paused"
	stateCancelled   = "cancelled"
	stateCompleted   = "completed"
	stateRollingBack = "rolling_back"
)

var defaultWavePercentages = []int32{1, 5, 25, 50, 100}

// computeWaves splits sentinel IDs into waves based on cumulative percentages.
// Each percentage defines the total number of sentinels that should be updated
// by the end of that wave. For example, [1, 5, 25, 50, 100] with 100 sentinels
// produces waves of sizes [1, 4, 20, 25, 50].
func computeWaves(sentinelIDs []string, percentages []int32) [][]string {
	total := len(sentinelIDs)
	waves := make([][]string, 0, len(percentages))
	assigned := 0

	for _, pct := range percentages {
		target := int(math.Ceil(float64(total) * float64(pct) / 100.0))
		if target > total {
			target = total
		}
		if target <= assigned {
			continue
		}
		waves = append(waves, sentinelIDs[assigned:target])
		assigned = target
	}

	return waves
}
