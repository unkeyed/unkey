package cron_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	freeTierExceededTemplate  = "free-tier-exceeded"
	freeTierThrottledTemplate = "free-tier-throttled"
)

type fakeAdmins struct {
	mu     sync.Mutex
	silent map[string]bool
}

func newFakeAdmins() *fakeAdmins {
	return &fakeAdmins{mu: sync.Mutex{}, silent: map[string]bool{}}
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

type recordingSender struct {
	mu   sync.Mutex
	sent []email.Email
}

func (r *recordingSender) Send(_ context.Context, e email.Email) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, e)
	return nil
}

func (r *recordingSender) count(templateID, to string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.sent {
		if e.TemplateID == templateID && len(e.To) == 1 && e.To[0] == to {
			n++
		}
	}
	return n
}

func TestFreeTierEnforcement_Integration(t *testing.T) {
	admins := newFakeAdmins()
	sender := &recordingSender{mu: sync.Mutex{}, sent: nil}
	h := harness.New(t, harness.WithAlertEmail(admins, sender))

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
			require.Equal(c, 1, sender.count(freeTierExceededTemplate, adminEmail(free.OrgID)))
		}, 30*time.Second, 200*time.Millisecond)
		require.Equal(t, 0, sender.count(freeTierExceededTemplate, adminEmail(paid.OrgID)))
		require.False(t, requestsPerMinute(t, free).Valid, "first breach must not throttle")
	})

	t.Run("warning is sent once while a throttle is pending", func(t *testing.T) {
		ws := freeWorkspace()

		first := enforce(t, ws)
		require.NotZero(t, first.GetWarnedAt())
		second := enforce(t, ws)
		require.Equal(t, first.GetWarnedAt(), second.GetWarnedAt())

		require.Equal(t, 1, sender.count(freeTierExceededTemplate, adminEmail(ws.OrgID)))
	})

	t.Run("no throttle is scheduled when no admin can be emailed", func(t *testing.T) {
		ws := freeWorkspace()
		admins.silence(ws.OrgID)

		resp := enforce(t, ws)
		require.Zero(t, resp.GetWarnedAt())
		require.Equal(t, 0, sender.count(freeTierExceededTemplate, adminEmail(ws.OrgID)))
	})

	t.Run("throttle applies the limit and notifies once due", func(t *testing.T) {
		ws := freeWorkspace()
		warnedAt := enforce(t, ws).GetWarnedAt()
		require.NotZero(t, warnedAt)

		throttle(t, ws, warnedAt)

		require.Equal(t, sql.NullInt32{Int32: 5, Valid: true}, requestsPerMinute(t, ws))
		require.Equal(t, 1, sender.count(freeTierThrottledTemplate, adminEmail(ws.OrgID)))

		again := enforce(t, ws)
		require.Zero(t, again.GetWarnedAt(), "a throttled workspace has nothing pending")
		require.Equal(t, 1, sender.count(freeTierExceededTemplate, adminEmail(ws.OrgID)))
	})

	t.Run("stale throttle schedule is ignored", func(t *testing.T) {
		ws := freeWorkspace()
		warnedAt := enforce(t, ws).GetWarnedAt()
		require.NotZero(t, warnedAt)

		throttle(t, ws, warnedAt-1)

		require.False(t, requestsPerMinute(t, ws).Valid)
		require.Equal(t, 0, sender.count(freeTierThrottledTemplate, adminEmail(ws.OrgID)))
	})

	t.Run("upgrade during the grace period cancels the throttle", func(t *testing.T) {
		ws := freeWorkspace()
		warnedAt := enforce(t, ws).GetWarnedAt()
		require.NotZero(t, warnedAt)

		setTier(t, ws, "Pro")
		throttle(t, ws, warnedAt)

		require.False(t, requestsPerMinute(t, ws).Valid)
		require.Equal(t, 0, sender.count(freeTierThrottledTemplate, adminEmail(ws.OrgID)))
	})
}
