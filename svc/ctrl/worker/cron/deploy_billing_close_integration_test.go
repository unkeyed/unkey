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

// fakeUsageReader ignores the query window and returns whatever rows the test sets.
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

// fakePusher records pushes by customer. failFor exercises the defer path.
// Mutex because the close fans out concurrently.
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
		// Terminal: fail fast. A retried push would stall the test for minutes.
		return 0, restate.TerminalError(errors.New("simulated push failure"))
	}
	f.pushed[req.StripeCustomerID] = req
	return 4, nil // cpu, memory, disk, egress
}

func (f *fakePusher) get(customerID string) (billingmeter.PushRequest, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	req, ok := f.pushed[customerID]
	return req, ok
}

// fakeCloser returns configured drafts per subscription and records finalizes.
type fakeCloser struct {
	mu           sync.Mutex
	drafts       map[string][]invoicecloser.DraftInvoice
	finalized    []string
	failFinalize map[string]bool
}

func newFakeCloser() *fakeCloser {
	return &fakeCloser{drafts: map[string][]invoicecloser.DraftInvoice{}, failFinalize: map[string]bool{}}
}

func (f *fakeCloser) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.drafts = map[string][]invoicecloser.DraftInvoice{}
	f.finalized = nil
	f.failFinalize = map[string]bool{}
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
	if f.failFinalize[invoiceID] {
		return false, errors.New("simulated finalize failure")
	}
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

// seedBillableWorkspace marks a workspace as an active Deploy customer.
func seedBillableWorkspace(t *testing.T, h *harness.Harness, customerID, subscriptionID string) string {
	t.Helper()
	ws := h.Seed.CreateWorkspace(h.Ctx)
	var sub any = subscriptionID
	if subscriptionID == "" {
		sub = nil
	}
	_, err := h.DB.RW().ExecContext(
		h.Ctx,
		"UPDATE workspaces SET deploy_plan = ?, stripe_customer_id = ?, stripe_subscription_id = ? WHERE id = ?",
		"pro", customerID, sub, ws.ID,
	)
	require.NoError(t, err)
	return ws.ID
}

func TestDeployBillingClose_Integration(t *testing.T) {
	reader := &fakeUsageReader{} //nolint:exhaustruct // zero value is an empty reader
	pusher := newFakePusher()
	closer := newFakeCloser()

	h := harness.New(t, harness.WithDeployBilling(reader, pusher, closer))

	// Yesterday's month: End() is in the past.
	now := time.Now().UTC()
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	closedPeriod := firstOfThisMonth.AddDate(0, 0, -1).Format("2006-01")
	p, err := billingperiod.Parse(closedPeriod)
	require.NoError(t, err)
	wantTimestamp := p.End().Add(-time.Second).Unix()

	runClose := func(period string, periodEnd int64) (*hydrav1.RunDeployBillingCloseResponse, error) {
		return hydrav1.NewCronServiceIngressClient(h.Restate, period).
			RunDeployBillingClose().
			Request(h.Ctx, &hydrav1.RunDeployBillingCloseRequest{PeriodEnd: periodEnd})
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

		resp, err := runClose(closedPeriod, 0)
		require.NoError(t, err)
		require.Equal(t, int32(1), resp.GetWorkspacesPushed())
		require.Equal(t, int32(1), resp.GetInvoicesFinalized())

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
			{ID: proration, BillingReason: "subscription_update", PeriodEnd: p.End().Unix()},
			{ID: nextPeriod, BillingReason: "subscription_cycle", PeriodEnd: p.End().Unix() + 1},
		})

		resp, err := runClose(closedPeriod, 0)
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

		// Push failed: invoice must stay open. We assert that, not whether the
		// handler returns an error (that varies by revision).
		_, _ = runClose(closedPeriod, 0)
		require.False(t, closer.didFinalize(invoiceID), "a failed-push workspace must not be finalized")
	})

	t.Run("refuses to close a period that has not ended", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		_, err := runClose(now.Format("2006-01"), 0)
		require.Error(t, err)

		current, err := billingperiod.Parse(now.Format("2006-01"))
		require.NoError(t, err)
		_, err = runClose(now.Format("2006-01"), current.End().Unix())
		require.NoError(t, err)
	})

	t.Run("defers close when stripe subscription id is missing", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		customerID := uid.New("cus")
		wsID := seedBillableWorkspace(t, h, customerID, "")
		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: wsID, ResourceID: "r1", CPUSeconds: 3},
		})

		resp, err := runClose(closedPeriod, 0)
		require.NoError(t, err)
		require.Equal(t, int32(0), resp.GetInvoicesFinalized())
	})

	t.Run("defers one workspace on finalize failure without aborting the batch", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		okCustomer := uid.New("cus")
		okSub := uid.New("sub")
		okWS := seedBillableWorkspace(t, h, okCustomer, okSub)

		failCustomer := uid.New("cus")
		failSub := uid.New("sub")
		failWS := seedBillableWorkspace(t, h, failCustomer, failSub)

		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: okWS, ResourceID: "r1", CPUSeconds: 4},
			{WorkspaceID: failWS, ResourceID: "r2", CPUSeconds: 6},
		})

		okInvoice := uid.New("in")
		failInvoice := uid.New("in")
		closer.setDrafts(okSub, []invoicecloser.DraftInvoice{
			{ID: okInvoice, BillingReason: "subscription_cycle", PeriodEnd: p.End().Unix()},
		})
		closer.setDrafts(failSub, []invoicecloser.DraftInvoice{
			{ID: failInvoice, BillingReason: "subscription_cycle", PeriodEnd: p.End().Unix()},
		})
		closer.failFinalize[failInvoice] = true

		resp, err := runClose(closedPeriod, 0)
		require.NoError(t, err)
		require.Equal(t, int32(2), resp.GetWorkspacesPushed())
		require.Equal(t, int32(1), resp.GetInvoicesFinalized())
		require.True(t, closer.didFinalize(okInvoice))
		require.False(t, closer.didFinalize(failInvoice))
	})
}

func TestCloseDeployBillingWorkspace_Integration(t *testing.T) {
	reader := &fakeUsageReader{} //nolint:exhaustruct
	pusher := newFakePusher()
	closer := newFakeCloser()
	h := harness.New(t, harness.WithDeployBilling(reader, pusher, closer))

	now := time.Now().UTC()
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	closedPeriod := firstOfThisMonth.AddDate(0, 0, -1).Format("2006-01")
	p, err := billingperiod.Parse(closedPeriod)
	require.NoError(t, err)
	wantTimestamp := p.End().Add(-time.Second).Unix()

	t.Run("pushes one workspace and finalizes the requested invoice", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		customerID := uid.New("cus")
		subscriptionID := uid.New("sub")
		wsID := seedBillableWorkspace(t, h, customerID, subscriptionID)
		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: wsID, ResourceID: "r1", CPUSeconds: 9},
		})
		invoiceID := uid.New("in")

		_, err := hydrav1.NewCronServiceIngressClient(h.Restate, wsID).
			CloseDeployBillingWorkspace().
			Request(h.Ctx, &hydrav1.CloseDeployBillingWorkspaceRequest{
				Period:    closedPeriod,
				PeriodEnd: 0,
				InvoiceId: invoiceID,
			})
		require.NoError(t, err)

		req, ok := pusher.get(customerID)
		require.True(t, ok, "expected a push for the workspace")
		require.Equal(t, wantTimestamp, req.Timestamp)
		require.InDelta(t, 9.0, req.Values.CPUSeconds, 1e-9)
		require.True(t, closer.didFinalize(invoiceID))
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

		_, err := hydrav1.NewCronServiceIngressClient(h.Restate, wsID).
			CloseDeployBillingWorkspace().
			Request(h.Ctx, &hydrav1.CloseDeployBillingWorkspaceRequest{
				Period:    closedPeriod,
				PeriodEnd: 0,
				InvoiceId: invoiceID,
			})
		require.NoError(t, err)
		require.False(t, closer.didFinalize(invoiceID))
	})
}
