package quotacheck

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/uid"
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
		{"warning at stale cutoff reschedules", now.Add(-throttleAfter - warningStaleAfter).Unix(), freeTierActionRescheduleThrottle},
		{"warning past stale cutoff warns again", now.Add(-throttleAfter - warningStaleAfter - time.Second).Unix(), freeTierActionWarn},
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
	now := time.Date(2026, time.November, 1, 15, 0, 0, 0, time.UTC)
	current := billingperiod.From(now)
	previous := current.Prev()

	free := sql.NullString{String: "Free", Valid: true}
	unthrottled := sql.NullInt32{Int32: 0, Valid: false}
	changedAt := func(t time.Time) sql.NullInt64 { return sql.NullInt64{Int64: t.UnixMilli(), Valid: true} }
	never := sql.NullInt64{Int64: 0, Valid: false}

	row := func(tier sql.NullString, perMinute sql.NullInt32, billingUpdatedAt sql.NullInt64) db.GetWorkspacesForQuotaCheckByIDsRow {
		var ws db.GetWorkspacesForQuotaCheckByIDsRow
		ws.Tier = tier
		ws.RequestsPerMinute = perMinute
		ws.BillingUpdatedAt = billingUpdatedAt
		return ws
	}

	cases := []struct {
		name   string
		ws     db.GetWorkspacesForQuotaCheckByIDsRow
		period billingperiod.Period
		want   bool
	}{
		{"current period free workspace", row(free, unthrottled, changedAt(now)), current, true},
		{"paid workspace", row(sql.NullString{String: "Pro", Valid: true}, unthrottled, never), current, false},
		{"already throttled", row(free, sql.NullInt32{Int32: 5, Valid: true}, never), current, false},
		{"previous period, billing never changed", row(free, unthrottled, never), previous, true},
		{"previous period, billing unchanged since before it", row(free, unthrottled, changedAt(previous.Start().Add(-time.Hour))), previous, true},
		{"previous period, cancelled at period end", row(free, unthrottled, changedAt(current.Start())), previous, false},
		{"previous period, downgraded mid period", row(free, unthrottled, changedAt(previous.Start().Add(10*24*time.Hour))), previous, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, shouldEnforceFreeTier(tc.ws, tc.period, now))
		})
	}
}

type freeWorkspaceDB struct {
	db.Database
	row db.GetWorkspacesForQuotaCheckByIDsRow
}

func (f freeWorkspaceDB) GetWorkspacesForQuotaCheckByIDs(_ context.Context, _ []string) ([]db.GetWorkspacesForQuotaCheckByIDsRow, error) {
	return []db.GetWorkspacesForQuotaCheckByIDsRow{f.row}, nil
}

type staticAdmins []string

func (a staticAdmins) AdminEmails(_ context.Context, _ string) ([]string, error) {
	return a, nil
}

var errSendFailed = errors.New("send failed")

type failingSender struct{}

func (failingSender) Send(_ context.Context, _ email.Email) error {
	return errSendFailed
}

func executeRun[O any](fn func(restate.RunContext) (any, error), output any, _ ...O) restate.TerminalError {
	value, err := fn(nil)
	if err != nil {
		return restate.ToTerminalError(err)
	}
	if value != nil {
		reflect.ValueOf(output).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func TestHandleEnforceFreeTierSchedulesNoThrottleWithoutDeliveredWarning(t *testing.T) {
	cases := []struct {
		name     string
		delivery *AlertDelivery
		wantErr  error
	}{
		{"delivery off", nil, nil},
		{"no recipients", &AlertDelivery{Admins: staticAdmins(nil), Email: email.NewNoop()}, nil},
		{"send fails", &AlertDelivery{Admins: staticAdmins{"admin@example.com"}, Email: failingSender{}}, errSendFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var row db.GetWorkspacesForQuotaCheckByIDsRow
			row.ID = uid.New(uid.WorkspacePrefix)
			row.Tier = sql.NullString{String: freeTierName, Valid: true}
			row.RequestsPerMonth = sql.NullInt64{Int64: 150_000, Valid: true}

			h := &Handler{
				db:              freeWorkspaceDB{Database: nil, row: row},
				clickhouse:      nil,
				heartbeat:       nil,
				slackWebhookURL: "",
				delivery:        tc.delivery,
				billingBaseURL:  "",
			}

			mockCtx := mocks.NewMockContext(t)
			mockCtx.EXPECT().Run(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(executeRun)
			mockCtx.EXPECT().Run(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(executeRun)
			mockCtx.EXPECT().Get(stateKeyFreeTierWarnedAt, mock.Anything).Return(false, nil)

			resp, err := h.HandleEnforceFreeTier(restate.WithMockContext(mockCtx), &hydrav1.EnforceFreeTierRequest{
				WorkspaceId: row.ID,
				Used:        200_000,
			})
			if tc.wantErr != nil {
				require.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
				require.Zero(t, resp.GetWarnedAt())
			}
			mockCtx.AssertNotCalled(t, "Set", stateKeyFreeTierWarnedAt, mock.Anything)
			mockCtx.AssertNotCalled(t, "Object", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
