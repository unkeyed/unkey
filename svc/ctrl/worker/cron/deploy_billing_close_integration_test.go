package cron_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/internal/invoicecloser"
)

// fakeUsageReader returns a fixed set of meter rows regardless of the query
// window, so a test controls exactly which workspaces have usage to push.
type fakeUsageReader struct {
	mu   sync.Mutex
	rows []clickhouse.InstanceMeterUsage
}

func (f *fakeUsageReader) set(rows []clickhouse.InstanceMeterUsage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = rows
}

func (f *fakeUsageReader) GetInstanceMeterUsage(
	_ context.Context,
	_ clickhouse.GetInstanceMeterUsageRequest,
) ([]clickhouse.InstanceMeterUsage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows, nil
}

// GetActiveKeysUsage returns no active-keys rows: this suite asserts the
// instance meters and the finalize behavior, not the active-keys meter.
func (f *fakeUsageReader) GetActiveKeysUsage(
	_ context.Context,
	_ clickhouse.GetActiveKeysUsageRequest,
) ([]clickhouse.ActiveKeysUsage, error) {
	return nil, nil
}

// fakePusher records the meter totals it is asked to push, keyed by customer,
// and can be told to fail for a given customer to exercise the defer path. The
// push fans out concurrently, so every field is mutex-guarded.
type fakePusher struct {
	mu      sync.Mutex
	pushed  map[string]billingmeter.PushRequest
	failFor map[string]bool
}

func newFakePusher() *fakePusher {
	return &fakePusher{pushed: map[string]billingmeter.PushRequest{}, failFor: map[string]bool{}}
}

func (f *fakePusher) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pushed = map[string]billingmeter.PushRequest{}
	f.failFor = map[string]bool{}
}

func (f *fakePusher) Push(_ context.Context, req billingmeter.PushRequest) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failFor[req.StripeCustomerID] {
		// Terminal so the push fails immediately instead of being retried: the
		// fanout close runs the push as its own Restate invocation bounded by a
		// multi-minute retry window, which a test must not wait out. The close's
		// defer-on-failure outcome is identical either way.
		return 0, restate.TerminalError(errors.New("simulated push failure"))
	}
	f.pushed[req.StripeCustomerID] = req
	return 4, nil // pretend four meter events were sent
}

func (f *fakePusher) get(customerID string) (billingmeter.PushRequest, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	req, ok := f.pushed[customerID]
	return req, ok
}

// fakeCloser serves configured draft invoices per subscription and records
// every invoice it finalizes. Like the pusher, it is hit concurrently.
type fakeCloser struct {
	mu        sync.Mutex
	drafts    map[string][]invoicecloser.DraftInvoice
	finalized []string
}

func newFakeCloser() *fakeCloser {
	return &fakeCloser{drafts: map[string][]invoicecloser.DraftInvoice{}}
}

func (f *fakeCloser) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.drafts = map[string][]invoicecloser.DraftInvoice{}
	f.finalized = nil
}

func (f *fakeCloser) setDrafts(subscriptionID string, drafts []invoicecloser.DraftInvoice) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.drafts[subscriptionID] = drafts
}

func (f *fakeCloser) ListDraftInvoices(_ context.Context, subscriptionID string) ([]invoicecloser.DraftInvoice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.drafts[subscriptionID], nil
}

func (f *fakeCloser) FinalizeInvoice(_ context.Context, invoiceID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finalized = append(f.finalized, invoiceID)
	return false, nil
}

func (f *fakeCloser) didFinalize(invoiceID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range f.finalized {
		if id == invoiceID {
			return true
		}
	}
	return false
}

// seedBillableWorkspace creates a workspace and marks it as an active Deploy
// customer (deploy_plan + stripe_customer_id + stripe_subscription_id), the
// shape ListDeployBillableWorkspaces selects and the close finalizes.
func seedBillableWorkspace(t *testing.T, h *harness.Harness, customerID, subscriptionID string) string {
	t.Helper()
	ws := h.Seed.CreateWorkspace(h.Ctx)
	_, err := h.DB.RW().ExecContext(
		h.Ctx,
		"UPDATE workspaces SET deploy_plan = ?, stripe_customer_id = ?, stripe_subscription_id = ? WHERE id = ?",
		"pro", customerID, subscriptionID, ws.ID,
	)
	require.NoError(t, err)
	return ws.ID
}

// TestDeployBillingClose_Integration drives RunDeployBillingClose end to end
// through Restate against fake usage/push/close dependencies, asserting the
// month-end close pushes the full-period total and finalizes only the right
// drafts.
func TestDeployBillingClose_Integration(t *testing.T) {
	reader := &fakeUsageReader{} //nolint:exhaustruct // zero value is an empty reader
	pusher := newFakePusher()
	closer := newFakeCloser()

	h := harness.New(t, harness.WithDeployBilling(reader, pusher, closer))

	// A period that has already ended: start of this month minus a day lands in
	// the previous month, whose End() is therefore in the past.
	now := time.Now().UTC()
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	closedPeriod := firstOfThisMonth.AddDate(0, 0, -1).Format("2006-01")
	p, err := billingperiod.Parse(closedPeriod)
	require.NoError(t, err)
	wantTimestamp := p.End().Add(-time.Second).Unix()

	runClose := func(period string) (*hydrav1.RunDeployBillingCloseResponse, error) {
		return hydrav1.NewCronServiceIngressClient(h.Restate, period).
			RunDeployBillingClose().
			Request(h.Ctx, &hydrav1.RunDeployBillingCloseRequest{})
	}

	t.Run("pushes full-period usage and finalizes the ended renewal invoice", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		customerID := uid.New("cus")
		subscriptionID := uid.New("sub")
		wsID := seedBillableWorkspace(t, h, customerID, subscriptionID)
		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: wsID, ResourceID: "r1", CPUSeconds: 12, MemoryGiBHours: 2, DiskGiBHours: 1, EgressBytes: 1 << 30},
		})
		invoiceID := uid.New("in")
		closer.setDrafts(subscriptionID, []invoicecloser.DraftInvoice{
			{ID: invoiceID, BillingReason: "subscription_cycle", PeriodEnd: p.End().Unix()},
		})

		resp, err := runClose(closedPeriod)
		require.NoError(t, err)
		require.Equal(t, int32(1), resp.GetWorkspacesPushed())
		require.Equal(t, int32(1), resp.GetInvoicesFinalized())

		// The push carries the workspace's absolute total, stamped one second
		// before the period ends so the "last"-formula meters bill the full month.
		req, ok := pusher.get(customerID)
		require.True(t, ok, "expected a push for the billable customer")
		require.Equal(t, wantTimestamp, req.Timestamp)
		require.InDelta(t, 12.0, req.Values.CPUSeconds, 1e-9)
		require.InDelta(t, 2.0*3600, req.Values.MemoryGiBSeconds, 1e-6)
		require.InDelta(t, 1.0, req.Values.EgressGiB, 1e-9)

		require.True(t, closer.didFinalize(invoiceID), "expected the ended cycle invoice to be finalized")
	})

	t.Run("skips proration and next-period drafts", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		customerID := uid.New("cus")
		subscriptionID := uid.New("sub")
		wsID := seedBillableWorkspace(t, h, customerID, subscriptionID)
		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: wsID, ResourceID: "r1", CPUSeconds: 5},
		})
		proration := uid.New("in")
		nextPeriod := uid.New("in")
		closer.setDrafts(subscriptionID, []invoicecloser.DraftInvoice{
			// Subscribe/upgrade proration: not a renewal, never finalized here.
			{ID: proration, BillingReason: "subscription_update", PeriodEnd: p.End().Unix()},
			// A renewal invoice whose period has not ended yet.
			{ID: nextPeriod, BillingReason: "subscription_cycle", PeriodEnd: p.End().Unix() + 1},
		})

		resp, err := runClose(closedPeriod)
		require.NoError(t, err)
		require.Equal(t, int32(0), resp.GetInvoicesFinalized())
		require.False(t, closer.didFinalize(proration))
		require.False(t, closer.didFinalize(nextPeriod))
	})

	t.Run("leaves the invoice open when the final push failed", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		customerID := uid.New("cus")
		subscriptionID := uid.New("sub")
		wsID := seedBillableWorkspace(t, h, customerID, subscriptionID)
		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: wsID, ResourceID: "r1", CPUSeconds: 7},
		})
		pusher.failFor[customerID] = true
		invoiceID := uid.New("in")
		closer.setDrafts(subscriptionID, []invoicecloser.DraftInvoice{
			{ID: invoiceID, BillingReason: "subscription_cycle", PeriodEnd: p.End().Unix()},
		})

		// The money-safety invariant: a workspace whose final push failed is
		// deferred, never finalized at an under-billed total. Whether the close
		// surfaces the deferral as an error or completes quietly is a version
		// detail (later revisions return a TerminalError so the backup cron
		// re-pushes), so assert the invariant, not the error.
		_, _ = runClose(closedPeriod)
		require.False(t, closer.didFinalize(invoiceID), "a failed-push workspace must not be finalized")
	})

	t.Run("refuses to close a period that has not ended", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		// The current month has not rolled over yet, so its End() is in the future.
		_, err := runClose(now.Format("2006-01"))
		require.Error(t, err)
	})
}
