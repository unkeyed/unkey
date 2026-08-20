// Package deploymentretention defines the deployment history kept by garbage
// collection. The cron scan and the per-deployment safety check share these
// values so they cannot apply different policies.
package deploymentretention

import "time"

const (
	ProductionAge = 14 * 24 * time.Hour
	PreviewAge    = 30 * 24 * time.Hour
	Successful    = int64(10)
)

// Cutoffs returns Unix-millisecond boundaries for the given collection time.
func Cutoffs(now time.Time) (production, preview int64) {
	return now.Add(-ProductionAge).UnixMilli(), now.Add(-PreviewAge).UnixMilli()
}
