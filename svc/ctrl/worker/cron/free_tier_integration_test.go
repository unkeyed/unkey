package cron_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

const (
	freeTierExceededTemplate  = "free-tier-exceeded"
	freeTierThrottledTemplate = "free-tier-throttled"
)

func TestFreeTierEnforcement_Integration(t *testing.T) {
	admins := newFakeAdmins()
	sender := email.NewCapture()
	h := harness.New(t, harness.WithAlertEmail(admins, sender))

	countEmails := func(templateID, to string) int {
		n := 0
		for _, e := range sender.Sent() {
			if e.TemplateID == templateID && len(e.To) == 1 && e.To[0] == to {
				n++
			}
		}
		return n
	}
	enforce := func(t *testing.T, ws db.Workspace) *hydrav1.EnforceFreeTierResponse {
		t.Helper()
		resp, err := hydrav1.NewCronServiceIngressClient(h.Restate, "free-tier-"+ws.ID).
			EnforceFreeTier().
			Request(h.Ctx, &hydrav1.EnforceFreeTierRequest{WorkspaceId: ws.ID, Used: 200_000})
		require.NoError(t, err)
		return resp
	}
	throttle := func(t *testing.T, ws db.Workspace, warnedAt int64) {
		t.Helper()
		_, err := hydrav1.NewCronServiceIngressClient(h.Restate, "free-tier-"+ws.ID).
			ThrottleFreeTierWorkspace().
			Request(h.Ctx, &hydrav1.ThrottleFreeTierWorkspaceRequest{WorkspaceId: ws.ID, WarnedAt: warnedAt})
		require.NoError(t, err)
	}
	requestsPerMinute := func(t *testing.T, ws db.Workspace) sql.NullInt32 {
		t.Helper()
		limits, err := h.DB.FindLimitsByWorkspaceID(h.Ctx, ws.ID)
		require.NoError(t, err)
		return limits.ApiRequestsCountMaxPerMinute
	}
	setTier := func(t *testing.T, ws db.Workspace, tier string) {
		t.Helper()
		require.NoError(t, h.DB.UpdateWorkspaceBillingTier(h.Ctx, db.UpdateWorkspaceBillingTierParams{
			Tier:        sql.NullString{String: tier, Valid: true},
			UpdatedAt:   sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			WorkspaceID: ws.ID,
		}))
	}
	freeWorkspace := func() db.Workspace {
		return h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 150_000})
	}

	t.Run("quota check warns free workspaces over quota but not paid ones", func(t *testing.T) {
		now := time.Now()
		free := freeWorkspace()
		paid := freeWorkspace()
		setTier(t, paid, "Pro")

		h.ClickHouseSeed.InsertVerifications(h.Ctx, free.ID, 200_000, now, "VALID")
		h.ClickHouseSeed.InsertVerifications(h.Ctx, paid.ID, 200_000, now, "VALID")
		waitForVerificationCount(t, h.Ctx, h.ClickHouse, free.ID, 200_000, now.Year(), int(now.Month()))
		waitForVerificationCount(t, h.Ctx, h.ClickHouse, paid.ID, 200_000, now.Year(), int(now.Month()))

		_, err := callRunQuotaCheck(h, now.Format("2006-01"))
		require.NoError(t, err)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			require.Equal(c, 1, countEmails(freeTierExceededTemplate, adminEmail(free.OrgID)))
		}, 30*time.Second, 200*time.Millisecond)
		require.Equal(t, 0, countEmails(freeTierExceededTemplate, adminEmail(paid.OrgID)))
		require.False(t, requestsPerMinute(t, free).Valid, "first breach must not throttle")
	})

	t.Run("warning is sent once while a throttle is pending", func(t *testing.T) {
		ws := freeWorkspace()

		first := enforce(t, ws)
		require.NotZero(t, first.GetWarnedAt())
		second := enforce(t, ws)
		require.Equal(t, first.GetWarnedAt(), second.GetWarnedAt())

		require.Equal(t, 1, countEmails(freeTierExceededTemplate, adminEmail(ws.OrgID)))
	})

	t.Run("no throttle is scheduled when no admin can be emailed", func(t *testing.T) {
		ws := freeWorkspace()
		admins.silence(ws.OrgID)

		resp := enforce(t, ws)
		require.Zero(t, resp.GetWarnedAt())
		require.Equal(t, 0, countEmails(freeTierExceededTemplate, adminEmail(ws.OrgID)))
	})

	t.Run("throttle applies the limit and notifies once due", func(t *testing.T) {
		ws := freeWorkspace()
		warnedAt := enforce(t, ws).GetWarnedAt()
		require.NotZero(t, warnedAt)

		throttle(t, ws, warnedAt)

		require.Equal(t, sql.NullInt32{Int32: 5, Valid: true}, requestsPerMinute(t, ws))
		require.Equal(t, 1, countEmails(freeTierThrottledTemplate, adminEmail(ws.OrgID)))

		again := enforce(t, ws)
		require.Zero(t, again.GetWarnedAt(), "a throttled workspace has nothing pending")
		require.Equal(t, 1, countEmails(freeTierExceededTemplate, adminEmail(ws.OrgID)))
	})

	t.Run("stale throttle schedule is ignored", func(t *testing.T) {
		ws := freeWorkspace()
		warnedAt := enforce(t, ws).GetWarnedAt()
		require.NotZero(t, warnedAt)

		throttle(t, ws, warnedAt-1)

		require.False(t, requestsPerMinute(t, ws).Valid)
		require.Equal(t, 0, countEmails(freeTierThrottledTemplate, adminEmail(ws.OrgID)))
	})

	t.Run("upgrade during the grace period cancels the throttle", func(t *testing.T) {
		ws := freeWorkspace()
		warnedAt := enforce(t, ws).GetWarnedAt()
		require.NotZero(t, warnedAt)

		setTier(t, ws, "Pro")
		throttle(t, ws, warnedAt)

		require.False(t, requestsPerMinute(t, ws).Valid)
		require.Equal(t, 0, countEmails(freeTierThrottledTemplate, adminEmail(ws.OrgID)))
	})
}

func TestFreeTierSlackFollowUpOnlyWhenThrottled_Integration(t *testing.T) {
	contextElements := func(messages ...slack.Payload) []slack.Text {
		var elements []slack.Text
		for _, message := range messages {
			for _, block := range message.Blocks {
				if block.Type == "context" {
					elements = append(elements, block.Elements...)
				}
			}
		}
		return elements
	}
	messages := make(chan slack.Payload, 10)
	var failNextSlack atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failNextSlack.Swap(false) {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var payload slack.Payload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		messages <- payload
	}))
	t.Cleanup(server.Close)
	sender := email.NewCapture()
	h := harness.New(t, harness.WithIsolatedMySQL(), harness.WithAlertEmail(newFakeAdmins(), sender), harness.WithSlackQuotaCheckWebhookURL(server.URL))
	free := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 150_000})
	paid := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 150_000})
	require.NoError(t, h.DB.UpdateWorkspaceBillingTier(h.Ctx, db.UpdateWorkspaceBillingTierParams{
		Tier: sql.NullString{String: "Pro", Valid: true}, WorkspaceID: paid.ID,
	}))
	now := time.Now().UTC()
	period := now.Format("2006-01")
	for _, ws := range []db.Workspace{free, paid} {
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 200_000, now, "VALID")
		waitForVerificationCount(t, h.Ctx, h.ClickHouse, ws.ID, 200_000, now.Year(), int(now.Month()))
	}

	_, err := callRunQuotaCheck(h, period)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	initial := []slack.Payload{<-messages, <-messages}
	for _, message := range initial {
		require.Contains(t, message.Text, "200,000 / 150,000 requests in "+now.Format("January 2006")+", 33.3% over quota")
		require.Equal(t, "Monthly request quota exceeded", message.Blocks[0].Text.Text)
	}
	for _, ws := range []db.Workspace{free, paid} {
		require.Contains(t, contextElements(initial...), slack.Text{Type: "mrkdwn", Text: "Workspace ID: `" + ws.ID + "`", Verbatim: true})
		require.Contains(t, contextElements(initial...), slack.Text{Type: "mrkdwn", Text: "Organization ID: `" + ws.OrgID + "`", Verbatim: true})
	}
	client := hydrav1.NewCronServiceIngressClient(h.Restate, "free-tier-"+free.ID)
	_, err = client.EnforceFreeTier().Request(h.Ctx, &hydrav1.EnforceFreeTierRequest{WorkspaceId: free.ID, Used: 200_000})
	require.NoError(t, err)
	require.Equal(t, 1, sender.CountByTemplate(freeTierExceededTemplate))

	setCronState(t, h, "quota-check-"+period, "notified_workspaces", map[string]int64{
		free.ID: now.Add(-8 * 24 * time.Hour).Unix(),
		paid.ID: now.Add(-8 * 24 * time.Hour).Unix(),
	})
	_, err = callRunQuotaCheck(h, period)
	require.NoError(t, err)
	require.Len(t, messages, 1, "only paid workspaces receive weekly reminders")
	reminder := <-messages
	require.Contains(t, reminder.Text, "Weekly reminder")
	require.Contains(t, reminder.Text, paid.Name)
	require.Equal(t, "Monthly request quota still exceeded", reminder.Blocks[0].Text.Text)
	require.Contains(t, reminder.Text, "200,000 / 150,000 requests in "+now.Format("January 2006")+", 33.3% over quota")
	require.Zero(t, sender.CountByTemplate(freeTierThrottledTemplate))

	warnedAt := now.Add(-13 * 24 * time.Hour).Unix()
	setCronState(t, h, "free-tier-"+free.ID, "warned_at", warnedAt)
	failNextSlack.Store(true)
	resp, err := client.EnforceFreeTier().Request(h.Ctx, &hydrav1.EnforceFreeTierRequest{WorkspaceId: free.ID, Used: 200_000})
	require.NoError(t, err)
	require.Equal(t, warnedAt, resp.GetWarnedAt(), "overdue enforcement must preserve the original warning")
	require.Equal(t, 1, sender.CountByTemplate(freeTierExceededTemplate))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		require.Equal(c, 1, sender.CountByTemplate(freeTierThrottledTemplate))
		require.Len(c, messages, 1, "Slack follows the actual throttle, not a later quota check")
	}, 30*time.Second, 100*time.Millisecond)
	message := <-messages
	require.Contains(t, message.Text, "limited to at most 5 per minute")
	require.Contains(t, message.Text, free.Name)
	require.Equal(t, "Free-tier API requests rate limited", message.Blocks[0].Text.Text)
	require.Contains(t, contextElements(message), slack.Text{Type: "mrkdwn", Text: "Workspace ID: `" + free.ID + "`", Verbatim: true})
	require.Contains(t, contextElements(message), slack.Text{Type: "mrkdwn", Text: "Organization ID: `" + free.OrgID + "`", Verbatim: true})
	limits, err := h.DB.FindLimitsByWorkspaceID(h.Ctx, free.ID)
	require.NoError(t, err)
	require.Equal(t, sql.NullInt32{Int32: 5, Valid: true}, limits.ApiRequestsCountMaxPerMinute)

	_, err = client.ThrottleFreeTierWorkspace().Request(h.Ctx, &hydrav1.ThrottleFreeTierWorkspaceRequest{WorkspaceId: free.ID, WarnedAt: warnedAt})
	require.NoError(t, err)
	setCronState(t, h, "quota-check-"+period, "notified_workspaces", map[string]int64{paid.ID: now.Unix()})
	_, err = callRunQuotaCheck(h, period)
	require.NoError(t, err)
	require.Empty(t, messages, "throttled workspaces must stay silent even after notification state resets")
	require.Equal(t, 1, sender.CountByTemplate(freeTierThrottledTemplate))

	upgraded := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 150_000})
	upgradedClient := hydrav1.NewCronServiceIngressClient(h.Restate, "free-tier-"+upgraded.ID)
	warning, err := upgradedClient.EnforceFreeTier().Request(h.Ctx, &hydrav1.EnforceFreeTierRequest{WorkspaceId: upgraded.ID, Used: 200_000})
	require.NoError(t, err)
	require.NotZero(t, warning.GetWarnedAt())
	require.NoError(t, h.DB.UpdateWorkspaceBillingTier(h.Ctx, db.UpdateWorkspaceBillingTierParams{
		Tier: sql.NullString{String: "Pro", Valid: true}, WorkspaceID: upgraded.ID,
	}))
	_, err = upgradedClient.ThrottleFreeTierWorkspace().Request(h.Ctx, &hydrav1.ThrottleFreeTierWorkspaceRequest{WorkspaceId: upgraded.ID, WarnedAt: warning.GetWarnedAt()})
	require.NoError(t, err)
	require.Empty(t, messages, "upgrading cancels the throttle follow-up")
	require.Equal(t, 1, sender.CountByTemplate(freeTierThrottledTemplate))
}

func TestFreeTierEnforcementDeliveryDisabled_Integration(t *testing.T) {
	h := harness.New(t)
	ws := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 150_000})

	resp, err := hydrav1.NewCronServiceIngressClient(h.Restate, "free-tier-"+ws.ID).
		EnforceFreeTier().
		Request(h.Ctx, &hydrav1.EnforceFreeTierRequest{WorkspaceId: ws.ID, Used: 200_000})
	require.NoError(t, err)
	require.Zero(t, resp.GetWarnedAt())
	limits, err := h.DB.FindLimitsByWorkspaceID(h.Ctx, ws.ID)
	require.NoError(t, err)
	require.False(t, limits.ApiRequestsCountMaxPerMinute.Valid)
}

func TestFreeTierEnforcementSendFailureDoesNotStartGracePeriod_Integration(t *testing.T) {
	capture := email.NewCapture()
	sender := &recoveringSender{sender: capture}
	sender.failing.Store(true)
	h := harness.New(t, harness.WithAlertEmail(newFakeAdmins(), sender))
	ws := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 150_000})
	client := hydrav1.NewCronServiceIngressClient(h.Restate, "free-tier-"+ws.ID)
	req := &hydrav1.EnforceFreeTierRequest{WorkspaceId: ws.ID, Used: 200_000}

	_, err := client.EnforceFreeTier().Request(h.Ctx, req)
	require.Error(t, err)
	require.Empty(t, capture.Sent())
	limits, err := h.DB.FindLimitsByWorkspaceID(h.Ctx, ws.ID)
	require.NoError(t, err)
	require.False(t, limits.ApiRequestsCountMaxPerMinute.Valid)

	sender.failing.Store(false)
	resp, err := client.EnforceFreeTier().Request(h.Ctx, req)
	require.NoError(t, err)
	require.NotZero(t, resp.GetWarnedAt())
	require.Equal(t, 1, capture.CountByTemplate(freeTierExceededTemplate))
}

func setCronState[T any](t *testing.T, h *harness.Harness, objectKey, stateKey string, value T) {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	state := make([]int, len(encoded))
	for i, b := range encoded {
		state[i] = int(b)
	}
	body, err := json.Marshal(struct {
		ObjectKey string           `json:"object_key"`
		NewState  map[string][]int `json:"new_state"`
	}{ObjectKey: objectKey, NewState: map[string][]int{stateKey: state}})
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(h.Ctx, http.MethodPost, h.RestateAdmin+"/services/hydra.v1.CronService/state", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
}

type fakeAdmins struct {
	mu     sync.Mutex
	silent map[string]bool
}

func newFakeAdmins() *fakeAdmins {
	return &fakeAdmins{silent: map[string]bool{}}
}

func (f *fakeAdmins) silence(orgID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.silent[orgID] = true
}

func (f *fakeAdmins) AdminEmails(_ context.Context, orgID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.silent[orgID] {
		return nil, nil
	}
	return []string{adminEmail(orgID)}, nil
}

func adminEmail(orgID string) string {
	return orgID + "@admins.test"
}

type recoveringSender struct {
	sender  email.Sender
	failing atomic.Bool
}

func (s *recoveringSender) Send(ctx context.Context, e email.Email) error {
	if s.failing.Load() {
		return errors.New("send failed")
	}
	return s.sender.Send(ctx, e)
}
