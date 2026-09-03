package deployusage

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenerateUsage(t *testing.T) {
	now := time.Date(2026, time.September, 3, 15, 42, 0, 0, time.UTC)
	targets := testTargets()

	rows := generateUsage(now, targets)
	start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	expectedHours := int(math.Ceil(now.Sub(start).Hours()))
	require.Len(t, rows, expectedHours*len(appProfiles)*2)
	require.Equal(t, start, rows[0].time)
	require.Equal(t, now.Truncate(time.Hour), rows[len(rows)-1].time)
	require.Equal(t, int64(168), rows[len(rows)-1].samplePairs)

	apps := make(map[string]struct{})
	environments := make(map[string]struct{})
	for _, row := range rows {
		apps[row.appID] = struct{}{}
		environments[row.environmentID] = struct{}{}
		require.Greater(t, row.cpuSeconds, 0.0)
		require.Greater(t, row.memoryGiBHours, 0.0)
		require.Greater(t, row.egressPublicBytes, int64(0))
	}
	require.Len(t, apps, 15)
	require.Len(t, environments, 30)
	require.Equal(t, rows, generateUsage(now, targets))
}

func TestLinkRedirectorIncidentResolves(t *testing.T) {
	profile := appProfiles[1]
	now := time.Date(2026, time.September, 3, 15, 42, 0, 0, time.UTC)
	before := dailyEgressGiB(time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC), now, profile, 1)
	peak := dailyEgressGiB(time.Date(2026, time.August, 28, 0, 0, 0, 0, time.UTC), now, profile, 1)
	after := dailyEgressGiB(time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), now, profile, 1)

	require.Less(t, before, 1.0)
	require.Equal(t, 435.0, peak)
	require.Less(t, after, 1.0)
}

func testTargets() []target {
	targets := make([]target, 0, len(appProfiles))
	for _, profile := range appProfiles {
		targets = append(targets, target{
			profile:      profile,
			appID:        profile.slug,
			productionID: profile.slug + "-production",
			previewID:    profile.slug + "-preview",
			workspaceID:  "ws_test",
			projectID:    "project_test",
		})
	}
	return targets
}
