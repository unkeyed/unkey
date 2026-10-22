package quotacheck

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestNextFreeTierAction(t *testing.T) {
	now := time.Date(2026, 10, 20, 16, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		warnedAt int64
		want     freeTierAction
	}{
		{"first breach warns", 0, freeTierActionWarn},
		{"warned today waits for scheduled throttle", now.Unix(), freeTierActionNone},
		{"throttle due today waits for scheduled throttle", now.Add(-throttleAfter).Unix(), freeTierActionNone},
		{"throttle at stall timeout waits", now.Add(-throttleAfter - throttleStallTimeout).Unix(), freeTierActionNone},
		{"throttle past stall timeout reschedules", now.Add(-throttleAfter - throttleStallTimeout - time.Second).Unix(), freeTierActionRescheduleThrottle},
		{"twelve day old warning reschedules", now.Add(-12 * 24 * time.Hour).Unix(), freeTierActionRescheduleThrottle},
		{"thirteen day old warning reschedules without another warning", now.Add(-13 * 24 * time.Hour).Unix(), freeTierActionRescheduleThrottle},
		{"month old warning reschedules without another warning", now.Add(-30 * 24 * time.Hour).Unix(), freeTierActionRescheduleThrottle},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, nextFreeTierAction(tc.warnedAt, now))
		})
	}
}

func TestIsFreeTier(t *testing.T) {
	require.True(t, isFreeTier(sql.NullString{String: "Free", Valid: true}))
	require.False(t, isFreeTier(sql.NullString{String: "", Valid: false}))
	require.False(t, isFreeTier(sql.NullString{String: "Pro Max", Valid: true}))
}

func TestIsThrottled(t *testing.T) {
	require.False(t, isThrottled(sql.NullInt32{Int32: 0, Valid: false}))
	require.True(t, isThrottled(sql.NullInt32{Int32: 0, Valid: true}))
	require.True(t, isThrottled(sql.NullInt32{Int32: throttleRequestsPerMinute, Valid: true}))
	require.False(t, isThrottled(sql.NullInt32{Int32: 1_000, Valid: true}))
}

func TestPreviousPeriodToRecheck(t *testing.T) {
	period := func(year int, month time.Month) billingperiod.Period {
		return billingperiod.Period{Year: year, Month: month}
	}

	cases := []struct {
		name   string
		period billingperiod.Period
		now    time.Time
		want   billingperiod.Period
		wantOK bool
	}{
		{"first day re-checks previous month", period(2026, time.November), time.Date(2026, time.November, 1, 15, 0, 0, 0, time.UTC), period(2026, time.October), true},
		{"first day of year re-checks december", period(2027, time.January), time.Date(2027, time.January, 1, 15, 0, 0, 0, time.UTC), period(2026, time.December), true},
		{"second day does not re-check", period(2026, time.November), time.Date(2026, time.November, 2, 15, 0, 0, 0, time.UTC), billingperiod.Period{}, false},
		{"previous period run does not chain", period(2026, time.October), time.Date(2026, time.November, 1, 15, 0, 0, 0, time.UTC), billingperiod.Period{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := previousPeriodToRecheck(tc.period, tc.now)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestShouldEnforceFreeTier(t *testing.T) {
	free := sql.NullString{String: "Free", Valid: true}
	unthrottled := sql.NullInt32{Int32: 0, Valid: false}

	row := func(tier sql.NullString, perMinute sql.NullInt32) db.GetWorkspacesForQuotaCheckByIDsRow {
		var ws db.GetWorkspacesForQuotaCheckByIDsRow
		ws.Tier = tier
		ws.RequestsPerMinute = perMinute
		return ws
	}

	cases := []struct {
		name string
		ws   db.GetWorkspacesForQuotaCheckByIDsRow
		want bool
	}{
		{"free workspace", row(free, unthrottled), true},
		{"paid workspace", row(sql.NullString{String: "Pro", Valid: true}, unthrottled), false},
		{"already throttled", row(free, sql.NullInt32{Int32: 5, Valid: true}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, shouldEnforceFreeTier(tc.ws))
		})
	}
}
