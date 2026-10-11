package quotacheck

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
)

func TestFreeQuotaWarningSecondRunSameMonthSendsNothing(t *testing.T) {
	ws := freeWorkspace("Acme", "acme", 150_000)
	sender := &fakeEmail{}
	admins := fakeAdmins{emails: map[string][]string{ws.OrgID: {"admin@example.com"}}}
	h := newWarningHandler(t, "live", true, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 160_000}, admins, sender)
	ctx, journal := statefulContext(t, "quota-check-2026-10")

	_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Len(t, sender.sent, 1)
	require.Equal(t, "free-quota-warning/"+ws.ID+"/2026-10", sender.sent[0].IdempotencyKey)
	require.Contains(t, journal.sent(), ws.ID)

	_, err = h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Len(t, sender.sent, 1)
}

func TestFreeQuotaWarningNewMonthSendsAgain(t *testing.T) {
	ws := freeWorkspace("Acme", "acme", 150_000)
	sender := &fakeEmail{}
	admins := fakeAdmins{emails: map[string][]string{ws.OrgID: {"admin@example.com"}}}
	h := newWarningHandler(t, "live", true, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 160_000}, admins, sender)

	october, octoberJournal := statefulContext(t, "quota-check-2026-10")
	_, err := h.Handle(restate.WithMockContext(october), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Len(t, sender.sent, 1)

	november, novemberJournal := statefulContext(t, "quota-check-2026-11")
	_, err = h.Handle(restate.WithMockContext(november), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Len(t, sender.sent, 2)
	require.Equal(t, "free-quota-warning/"+ws.ID+"/2026-11", sender.sent[1].IdempotencyKey)
	require.Contains(t, octoberJournal.sent(), ws.ID)
	require.Contains(t, novemberJournal.sent(), ws.ID)
}

func TestFreeQuotaWarningFailedSendIsRetriedNextDay(t *testing.T) {
	ws := freeWorkspace("Acme", "acme", 150_000)
	sender := &fakeEmail{fail: 1}
	admins := fakeAdmins{emails: map[string][]string{ws.OrgID: {"admin@example.com"}}}
	h := newWarningHandler(t, "live", true, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 160_000}, admins, sender)
	ctx, journal := statefulContext(t, "quota-check-2026-10")
	logs := loggertest.Install(t)
	start := logs.Snapshot()

	_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Empty(t, sender.sent)
	require.NotContains(t, journal.sent(), ws.ID)
	require.NotEmpty(t, recordsWith(logs.Since(start), "free quota warning send failed"))

	_, err = h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Len(t, sender.sent, 1)
	require.Contains(t, journal.sent(), ws.ID)
}

func TestFreeQuotaWarningDryRunDoesNotMark(t *testing.T) {
	ws := freeWorkspace("Acme", "acme", 150_000)
	sender := &fakeEmail{}
	admins := fakeAdmins{emails: map[string][]string{
		ws.OrgID: {"one@example.com", "two@example.com"},
	}}
	usage := map[string]int64{ws.ID: 160_000}
	rows := []db.GetWorkspacesForQuotaCheckByIDsRow{ws}
	dry := newWarningHandler(t, "dry_run", true, rows, usage, admins, sender)
	ctx, journal := statefulContext(t, "quota-check-2026-10")
	logs := loggertest.Install(t)
	start := logs.Snapshot()

	_, err := dry.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Empty(t, sender.sent)
	require.NotContains(t, journal.sent(), ws.ID)
	requireNoEmails(t, logs.Since(start))

	dryLogs := recordsWith(logs.Since(start), "free quota warning dry run")
	require.Len(t, dryLogs, 1)
	attrs := loggertest.FlatAttrs(dryLogs[0])
	require.Equal(t, ws.ID, attrs["workspace_id"])
	require.Equal(t, "Acme", attrs["name"])
	require.Equal(t, "acme", attrs["slug"])
	require.EqualValues(t, 160_000, attrs["usage"])
	require.EqualValues(t, 150_000, attrs["cap"])
	require.EqualValues(t, 2, attrs["admin_count"])

	live := newWarningHandler(t, "live", true, rows, usage, admins, sender)
	_, err = live.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Len(t, sender.sent, 1)
	require.Equal(t, []string{"one@example.com", "two@example.com"}, sender.sent[0].To)
}

func TestFreeQuotaWarningFilters(t *testing.T) {
	free := freeWorkspace("Free Co", "free-co", 150_000)
	free.Tier = sql.NullString{}
	free.Subscriptions = mustJSON(t, map[string]string{})

	paid := freeWorkspace("Paid Co", "paid-co", 250_000)
	paid.Tier = sql.NullString{String: "API Pro 250k", Valid: true}

	subscribed := freeWorkspace("Sub Co", "sub-co", 150_000)
	subscribed.ApiSubscriptionID = sql.NullString{String: uid.New(uid.TestPrefix), Valid: true}

	disabled := freeWorkspace("Disabled Co", "disabled-co", 150_000)
	disabled.Enabled = false

	deleted := freeWorkspace("Deleted Co", "deleted-co", 150_000)
	deleted.DeletedAtM = sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}

	legacy := freeWorkspace("Legacy Co", "legacy-co", 150_000)
	legacy.Subscriptions = mustJSON(t, map[string]string{"plan": "custom"})

	missingCap := freeWorkspace("No Cap", "no-cap", 150_000)
	missingCap.RequestsPerMonth = sql.NullInt64{}

	rows := []db.GetWorkspacesForQuotaCheckByIDsRow{free, paid, subscribed, disabled, deleted, legacy, missingCap}
	usage := map[string]int64{}
	adminsByOrg := map[string][]string{}
	for _, row := range rows {
		// Above every cap in this set, including the paid workspace's 250,000.
		usage[row.ID] = 300_000
		adminsByOrg[row.OrgID] = []string{"admin@example.com"}
	}
	sender := &fakeEmail{}
	h := newWarningHandler(t, "dry_run", true, rows, usage, fakeAdmins{emails: adminsByOrg}, sender)
	ctx, journal := statefulContext(t, "quota-check-2026-10")
	logs := loggertest.Install(t)
	start := logs.Snapshot()

	_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Empty(t, sender.sent)
	require.NotContains(t, journal.sent(), free.ID)

	skipped := map[string]string{}
	for _, record := range recordsWith(logs.Since(start), "free quota warning skipped") {
		attrs := loggertest.FlatAttrs(record)
		id, _ := attrs["workspace_id"].(string)
		reason, _ := attrs["reason"].(string)
		skipped[id] = reason
	}
	require.Equal(t, "paid_tier", skipped[paid.ID])
	require.Equal(t, "api_subscription", skipped[subscribed.ID])
	require.Equal(t, "disabled", skipped[disabled.ID])
	require.Equal(t, "deleted", skipped[deleted.ID])
	require.Equal(t, "legacy_subscriptions", skipped[legacy.ID])
	require.Equal(t, "missing_cap", skipped[missingCap.ID])
	require.NotContains(t, skipped, free.ID)

	dry := recordsWith(logs.Since(start), "free quota warning dry run")
	require.Len(t, dry, 1)
	require.Equal(t, free.ID, loggertest.FlatAttrs(dry[0])["workspace_id"])

	// Paid, deleted, and legacy workspaces are still recorded for Slack.
	// The warning filters do not change that path.
	notified := journal.notified()
	require.Contains(t, notified, paid.ID)
	require.Contains(t, notified, deleted.ID)
	require.Contains(t, notified, legacy.ID)
	require.NotContains(t, notified, disabled.ID)
	require.NotContains(t, notified, missingCap.ID)
}

func TestFreeQuotaWarningSkipsConfiguredRateLimit(t *testing.T) {
	open := freeWorkspace("Open", "open", 150_000)
	throttled := freeWorkspace("Throttled", "throttled", 150_000)
	throttled.ApiRequestsCountMaxPerMinute = sql.NullInt32{Int32: 100, Valid: true}
	blocked := freeWorkspace("Blocked", "blocked", 150_000)
	blocked.ApiRequestsCountMaxPerMinute = sql.NullInt32{Int32: 0, Valid: true}

	rows := []db.GetWorkspacesForQuotaCheckByIDsRow{open, throttled, blocked}
	usage := map[string]int64{open.ID: 160_000, throttled.ID: 160_000, blocked.ID: 160_000}
	adminsByOrg := map[string][]string{}
	for _, row := range rows {
		adminsByOrg[row.OrgID] = []string{"admin@example.com"}
	}
	sender := &fakeEmail{}
	h := newWarningHandler(t, "dry_run", true, rows, usage, fakeAdmins{emails: adminsByOrg}, sender)
	ctx, journal := statefulContext(t, "quota-check-2026-10")
	logs := loggertest.Install(t)
	start := logs.Snapshot()

	_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Empty(t, sender.sent)
	require.NotContains(t, journal.sent(), open.ID)

	reasons := map[string]slog.Record{}
	for _, record := range recordsWith(logs.Since(start), "free quota warning skipped") {
		id, _ := loggertest.FlatAttrs(record)["workspace_id"].(string)
		reasons[id] = record
	}
	require.Equal(t, "rate_limited", loggertest.FlatAttrs(reasons[throttled.ID])["reason"])
	require.EqualValues(t, 100, loggertest.FlatAttrs(reasons[throttled.ID])["requests_per_minute"])
	require.Equal(t, "rate_limited", loggertest.FlatAttrs(reasons[blocked.ID])["reason"])
	require.EqualValues(t, 0, loggertest.FlatAttrs(reasons[blocked.ID])["requests_per_minute"])

	dry := recordsWith(logs.Since(start), "free quota warning dry run")
	require.Len(t, dry, 1)
	require.Equal(t, open.ID, loggertest.FlatAttrs(dry[0])["workspace_id"])

	liveSender := &fakeEmail{}
	live := newWarningHandler(t, "live", true, rows, usage, fakeAdmins{emails: adminsByOrg}, liveSender)
	liveCtx, liveJournal := statefulContext(t, "quota-check-2026-10")
	_, err = live.Handle(restate.WithMockContext(liveCtx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Len(t, liveSender.sent, 1)
	require.Equal(t, []string{"admin@example.com"}, liveSender.sent[0].To)
	require.Equal(t, freeQuotaWarningTemplate, liveSender.sent[0].TemplateID)
	require.Contains(t, liveJournal.sent(), open.ID)
	require.NotContains(t, liveJournal.sent(), throttled.ID)
	require.NotContains(t, liveJournal.sent(), blocked.ID)
}

func TestFreeQuotaWarningModes(t *testing.T) {
	t.Run("off sends nothing and does not touch the email map", func(t *testing.T) {
		ws := freeWorkspace("Acme", "acme", 150_000)
		sender := &fakeEmail{}
		h := newWarningHandler(t, "off", true, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 160_000}, fakeAdmins{emails: map[string][]string{ws.OrgID: {"admin@example.com"}}}, sender)
		ctx, journal := statefulContext(t, "quota-check-2026-10")
		logs := loggertest.Install(t)
		start := logs.Snapshot()

		_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
		require.NoError(t, err)
		require.Empty(t, sender.sent)
		require.NotContains(t, journal.gets, stateKeyFreeQuotaWarningSent)
		require.NotContains(t, journal.sent(), ws.ID)
		require.Contains(t, journal.notified(), ws.ID)
		require.Empty(t, recordsWith(logs.Since(start), "free quota warning dry run"))
		require.Empty(t, recordsWith(logs.Since(start), "free quota warning sent"))
	})

	t.Run("dry_run logs only", func(t *testing.T) {
		ws := freeWorkspace("Acme", "acme", 150_000)
		sender := &fakeEmail{}
		h := newWarningHandler(t, "dry_run", true, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 160_000}, fakeAdmins{emails: map[string][]string{ws.OrgID: {"admin@example.com"}}}, sender)
		ctx, journal := statefulContext(t, "quota-check-2026-10")
		logs := loggertest.Install(t)
		start := logs.Snapshot()

		_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
		require.NoError(t, err)
		require.Empty(t, sender.sent)
		require.NotContains(t, journal.sent(), ws.ID)
		require.Len(t, recordsWith(logs.Since(start), "free quota warning dry run"), 1)
		requireNoEmails(t, logs.Since(start))
	})

	t.Run("live sends and marks", func(t *testing.T) {
		ws := freeWorkspace("Acme", "acme", 150_000)
		sender := &fakeEmail{}
		h := newWarningHandler(t, "live", true, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 180_000}, fakeAdmins{emails: map[string][]string{ws.OrgID: {"admin@example.com"}}}, sender)
		ctx, journal := statefulContext(t, "quota-check-2026-10")

		_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
		require.NoError(t, err)
		require.Len(t, sender.sent, 1)
		require.Equal(t, freeQuotaWarningTemplate, sender.sent[0].TemplateID)
		require.Empty(t, sender.sent[0].From)
		require.Empty(t, sender.sent[0].Subject)
		require.Equal(t, "Acme", sender.sent[0].Variables["WORKSPACE_NAME"])
		require.Equal(t, "180,000", sender.sent[0].Variables["USED"])
		require.Equal(t, "150,000", sender.sent[0].Variables["LIMIT"])
		require.Equal(t, "https://app.unkey.com/acme/settings/billing", sender.sent[0].Variables["BILLING_URL"])
		require.Equal(t, fmt.Sprintf("%d", time.Now().Year()), sender.sent[0].Variables["YEAR"])
		require.Contains(t, journal.sent(), ws.ID)
	})

	t.Run("live without a resend key behaves as off", func(t *testing.T) {
		ws := freeWorkspace("Acme", "acme", 150_000)
		sender := &fakeEmail{}
		h := newWarningHandler(t, "live", false, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 160_000}, fakeAdmins{emails: map[string][]string{ws.OrgID: {"admin@example.com"}}}, sender)
		ctx, journal := statefulContext(t, "quota-check-2026-10")
		logs := loggertest.Install(t)
		start := logs.Snapshot()

		_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
		require.NoError(t, err)
		require.Empty(t, sender.sent)
		require.NotContains(t, journal.gets, stateKeyFreeQuotaWarningSent)
		require.NotContains(t, journal.sent(), ws.ID)
		require.Contains(t, journal.notified(), ws.ID)
		require.NotEmpty(t, recordsWith(logs.Since(start), "free API quota warning emails are DISABLED: quota_warning_email_mode is live but no resend api key is configured; behaving as off"))
	})
}

func TestFreeQuotaWarningUsageExactlyAtCapCounts(t *testing.T) {
	ws := freeWorkspace("Acme", "acme", 150_000)
	sender := &fakeEmail{}
	h := newWarningHandler(t, "live", true, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 150_000}, fakeAdmins{emails: map[string][]string{ws.OrgID: {"admin@example.com"}}}, sender)
	ctx, journal := statefulContext(t, "quota-check-2026-10")

	resp, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.GetWorkspacesExceeded())
	require.Len(t, sender.sent, 1)
	require.Equal(t, "150,000", sender.sent[0].Variables["USED"])
	require.Equal(t, "150,000", sender.sent[0].Variables["LIMIT"])
	require.Contains(t, journal.sent(), ws.ID)
}

func TestFreeQuotaWarningNoAdminsSkips(t *testing.T) {
	ws := freeWorkspace("Acme", "acme", 150_000)
	sender := &fakeEmail{}
	h := newWarningHandler(t, "live", true, []db.GetWorkspacesForQuotaCheckByIDsRow{ws}, map[string]int64{ws.ID: 160_000}, fakeAdmins{}, sender)
	ctx, journal := statefulContext(t, "quota-check-2026-10")
	logs := loggertest.Install(t)
	start := logs.Snapshot()

	_, err := h.Handle(restate.WithMockContext(ctx), &hydrav1.RunQuotaCheckRequest{})
	require.NoError(t, err)
	require.Empty(t, sender.sent)
	require.NotContains(t, journal.sent(), ws.ID)
	skipped := recordsWith(logs.Since(start), "free quota warning skipped")
	require.Len(t, skipped, 1)
	require.Equal(t, "no_admins", loggertest.FlatAttrs(skipped[0])["reason"])
	requireNoEmails(t, logs.Since(start))
}

func TestHasWorkspaceRateLimit(t *testing.T) {
	require.False(t, hasWorkspaceRateLimit(sql.NullInt32{}))
	require.True(t, hasWorkspaceRateLimit(sql.NullInt32{Int32: 0, Valid: true}))
	require.True(t, hasWorkspaceRateLimit(sql.NullInt32{Int32: 60, Valid: true}))
}

func TestHasLegacySubscriptions(t *testing.T) {
	require.False(t, hasLegacySubscriptions(nil))
	require.False(t, hasLegacySubscriptions(mustJSON(t, nil)))
	require.False(t, hasLegacySubscriptions(mustJSON(t, map[string]string{})))
	require.False(t, hasLegacySubscriptions(mustJSON(t, []string{})))
	require.True(t, hasLegacySubscriptions(mustJSON(t, map[string]string{"plan": "custom"})))
	require.True(t, hasLegacySubscriptions([]byte{0xff}))
}

type quotaDB struct {
	db.Database
	rows []db.GetWorkspacesForQuotaCheckByIDsRow
}

func (q quotaDB) GetWorkspacesForQuotaCheckByIDs(context.Context, []string) ([]db.GetWorkspacesForQuotaCheckByIDsRow, error) {
	return q.rows, nil
}

type quotaCH struct {
	clickhouse.ClickHouse
	usage map[string]int64
}

func (q quotaCH) GetBillableUsageAboveThreshold(context.Context, int, int, int64) (map[string]int64, error) {
	return q.usage, nil
}

type fakeAdmins struct {
	emails map[string][]string
}

func (f fakeAdmins) AdminEmails(_ context.Context, orgID string) ([]string, error) {
	return f.emails[orgID], nil
}

type fakeEmail struct {
	fail int
	sent []email.Email
}

func (f *fakeEmail) Send(_ context.Context, msg email.Email) error {
	if f.fail > 0 {
		f.fail--
		return errors.New("resend unavailable")
	}
	f.sent = append(f.sent, msg)
	return nil
}

type stateJournal struct {
	state map[string]any
	gets  []string
}

func (j *stateJournal) read(key string, output any) {
	j.gets = append(j.gets, key)
	value, ok := j.state[key]
	if !ok {
		return
	}
	reflect.ValueOf(output).Elem().Set(reflect.ValueOf(cloneState(value)))
}

func (j *stateJournal) write(key string, value any) {
	j.state[key] = cloneState(value)
}

func (j *stateJournal) sent() map[string]int64 {
	value, ok := j.state[stateKeyFreeQuotaWarningSent].(map[string]int64)
	if !ok {
		return map[string]int64{}
	}
	return value
}

func (j *stateJournal) notified() map[string]int64 {
	value, ok := j.state[stateKeyNotifiedWorkspaces].(map[string]int64)
	if !ok {
		return map[string]int64{}
	}
	return value
}

func cloneState(value any) any {
	src, ok := value.(map[string]int64)
	if !ok {
		return value
	}
	dst := make(map[string]int64, len(src))
	for key, item := range src {
		dst[key] = item
	}
	return dst
}

func statefulContext(t *testing.T, key string) (*mocks.MockContext, *stateJournal) {
	t.Helper()
	ctx := mocks.NewMockContext(t)
	journal := &stateJournal{state: map[string]any{}}

	keyCall := ctx.EXPECT().Key()
	keyCall.Return(key)
	keyCall.Maybe()

	getCall := ctx.EXPECT().Get(mock.Anything, mock.Anything)
	getCall.Call.Run(func(args mock.Arguments) {
		journal.read(args.String(0), args.Get(1))
	}).Return(true, nil)
	getCall.Maybe()

	setCall := ctx.EXPECT().Set(mock.Anything, mock.Anything)
	setCall.Call.Run(func(args mock.Arguments) {
		journal.write(args.String(0), args.Get(1))
	}).Return()
	setCall.Maybe()

	run := func(args mock.Arguments) { invokeRun(t, args) }
	one := ctx.EXPECT().Run(mock.Anything, mock.Anything, mock.Anything)
	one.Call.Run(run).Return(nil)
	one.Maybe()
	two := ctx.EXPECT().Run(mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	two.Call.Run(run).Return(nil)
	two.Maybe()

	return ctx, journal
}

func invokeRun(t *testing.T, args mock.Arguments) {
	t.Helper()
	fn := reflect.ValueOf(args.Get(0))
	results := fn.Call([]reflect.Value{reflect.Zero(fn.Type().In(0))})
	require.Len(t, results, 2)
	if !results[1].IsNil() {
		return
	}
	val := results[0]
	if val.Kind() == reflect.Interface {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	if !val.IsValid() || (nillable(val) && val.IsNil()) {
		return
	}
	out := reflect.ValueOf(args.Get(1))
	require.Equal(t, reflect.Pointer, out.Kind())
	out.Elem().Set(val)
}

func nillable(v reflect.Value) bool {
	kind := v.Kind()
	return kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Interface ||
		kind == reflect.Map ||
		kind == reflect.Pointer ||
		kind == reflect.Slice
}

func newWarningHandler(
	t *testing.T,
	mode string,
	resendConfigured bool,
	rows []db.GetWorkspacesForQuotaCheckByIDsRow,
	usage map[string]int64,
	admins workos.Resolver,
	sender email.Sender,
) *Handler {
	t.Helper()
	h, err := New(Config{
		DB:               quotaDB{Database: nil, rows: rows},
		Clickhouse:       quotaCH{ClickHouse: nil, usage: usage},
		Heartbeat:        healthcheck.NewNoop(),
		SlackWebhookURL:  "",
		EmailMode:        mode,
		ResendConfigured: resendConfigured,
		Admins:           admins,
		Email:            sender,
		BillingBaseURL:   "https://app.unkey.com",
	})
	require.NoError(t, err)
	return h
}

func freeWorkspace(name, slug string, cap int64) db.GetWorkspacesForQuotaCheckByIDsRow {
	return db.GetWorkspacesForQuotaCheckByIDsRow{
		ID:                           uid.New(uid.WorkspacePrefix),
		OrgID:                        uid.New(uid.OrgPrefix),
		Name:                         name,
		Slug:                         slug,
		Enabled:                      true,
		DeletedAtM:                   sql.NullInt64{},
		Subscriptions:                nil,
		StripeCustomerID:             sql.NullString{},
		Tier:                         sql.NullString{String: "Free", Valid: true},
		RequestsPerMonth:             sql.NullInt64{Int64: cap, Valid: true},
		ApiRequestsCountMaxPerMinute: sql.NullInt32{},
		ApiSubscriptionID:            sql.NullString{},
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return raw
}

func recordsWith(records []slog.Record, message string) []slog.Record {
	var matched []slog.Record
	for _, record := range records {
		if record.Message == message {
			matched = append(matched, record)
		}
	}
	return matched
}

func requireNoEmails(t *testing.T, records []slog.Record) {
	t.Helper()
	for _, record := range records {
		for key, value := range loggertest.FlatAttrs(record) {
			require.NotContains(t, fmt.Sprint(value), "@", "log attr %s leaked an email address", key)
		}
	}
}
