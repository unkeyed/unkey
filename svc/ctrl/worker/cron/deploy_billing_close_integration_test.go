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
	mu                         sync.Mutex
	rows                       []clickhouse.InstanceMeterUsage
	activeKeys                 []clickhouse.ActiveKeysUsage
	instanceReads              int
	activeKeyReads             int
	instanceReadDelay          time.Duration
	instanceReadErr            error
	activeInstanceReads        int
	maxConcurrentInstanceReads int
}

func (f *fakeUsageReader) set(rows []clickhouse.InstanceMeterUsage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = rows
	f.activeKeys = nil
	f.instanceReads = 0
	f.activeKeyReads = 0
	f.instanceReadDelay = 0
	f.instanceReadErr = nil
	f.activeInstanceReads = 0
	f.maxConcurrentInstanceReads = 0
}

func (f *fakeUsageReader) setActiveKeys(rows []clickhouse.ActiveKeysUsage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activeKeys = rows
}

func (f *fakeUsageReader) GetInstanceMeterUsage(
	ctx context.Context,
	req clickhouse.GetInstanceMeterUsageRequest,
) ([]clickhouse.InstanceMeterUsage, error) {
	f.mu.Lock()
	f.instanceReads++
	f.activeInstanceReads++
	f.maxConcurrentInstanceReads = max(f.maxConcurrentInstanceReads, f.activeInstanceReads)
	delay := f.instanceReadDelay
	readErr := f.instanceReadErr
	workspaceIDs := make(map[string]struct{}, len(req.WorkspaceIDs))
	for _, workspaceID := range req.WorkspaceIDs {
		workspaceIDs[workspaceID] = struct{}{}
	}
	rows := make([]clickhouse.InstanceMeterUsage, 0, len(f.rows))
	for _, row := range f.rows {
		_, included := workspaceIDs[row.WorkspaceID]
		if (req.WorkspaceID == "" || row.WorkspaceID == req.WorkspaceID) &&
			(req.WorkspaceIDs == nil || included) {
			rows = append(rows, row)
		}
	}
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.activeInstanceReads--
		f.mu.Unlock()
	}()

	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if readErr != nil {
		return nil, readErr
	}
	return rows, nil
}

func (f *fakeUsageReader) GetActiveKeysUsage(
	_ context.Context,
	req clickhouse.GetActiveKeysUsageRequest,
) ([]clickhouse.ActiveKeysUsage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activeKeyReads++
	workspaceIDs := make(map[string]struct{}, len(req.WorkspaceIDs))
	for _, workspaceID := range req.WorkspaceIDs {
		workspaceIDs[workspaceID] = struct{}{}
	}
	rows := make([]clickhouse.ActiveKeysUsage, 0, len(f.activeKeys))
	for _, row := range f.activeKeys {
		_, included := workspaceIDs[row.WorkspaceID]
		if (req.WorkspaceID == "" || row.WorkspaceID == req.WorkspaceID) &&
			(req.WorkspaceIDs == nil || included) {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (f *fakeUsageReader) reads() (instance, activeKeys int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.instanceReads, f.activeKeyReads
}

func (f *fakeUsageReader) trackInstanceConcurrency(delay time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.instanceReadDelay = delay
	f.maxConcurrentInstanceReads = 0
}

func (f *fakeUsageReader) failInstanceReads(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.instanceReadErr = err
}

func (f *fakeUsageReader) maxInstanceConcurrency() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxConcurrentInstanceReads
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
	claimed      map[string]int64
	failFinalize map[string]bool
}

func newFakeCloser() *fakeCloser {
	return &fakeCloser{
		drafts:       map[string][]invoicecloser.DraftInvoice{},
		claimed:      map[string]int64{},
		failFinalize: map[string]bool{},
	}
}

func (f *fakeCloser) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.drafts = map[string][]invoicecloser.DraftInvoice{}
	f.finalized = nil
	f.claimed = map[string]int64{}
	f.failFinalize = map[string]bool{}
}

// claimedAt reports the finalization deadline the close pushed onto an invoice,
// or 0 if it was never claimed. The claim has to happen before the ingestion
// wait, or Stripe's own ~1h auto-finalize closes the invoice while the close is
// still sleeping.
func (f *fakeCloser) claimedAt(invoiceID string) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.claimed[invoiceID]
}

func (f *fakeCloser) ClaimInvoice(_ context.Context, invoiceID string, finalizeAt int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claimed[invoiceID] = finalizeAt
	return nil
}

func (f *fakeCloser) GetInvoice(_ context.Context, invoiceID string) (invoicecloser.DraftInvoice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, drafts := range f.drafts {
		for _, d := range drafts {
			if d.ID == invoiceID {
				return d, nil
			}
		}
	}
	return invoicecloser.DraftInvoice{}, invoicecloser.ErrNotFound //nolint:exhaustruct // zero value on the not-found path
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

// seedBillableWorkspace marks a workspace as an active Deploy customer. The
// Deploy subscription id lives on billing_subscriptions, not
// workspace_billing, so it is only inserted there, and only when non-empty
// (some tests exercise the no-subscription-id defer path).
func seedBillableWorkspace(t *testing.T, h *harness.Harness, customerID, subscriptionID string) string {
	t.Helper()
	ws := h.Seed.CreateWorkspace(h.Ctx)
	_, err := h.DB.RW().ExecContext(
		h.Ctx,
		"UPDATE workspace_billing SET plan = ?, stripe_customer_id = ? WHERE workspace_id = ?",
		"pro", customerID, ws.ID,
	)
	require.NoError(t, err)
	if subscriptionID != "" {
		_, err = h.DB.RW().ExecContext(
			h.Ctx,
			"INSERT INTO billing_subscriptions (workspace_id, product, stripe_subscription_id) VALUES (?, 'compute', ?)",
			ws.ID, subscriptionID,
		)
		require.NoError(t, err)
	}
	return ws.ID
}

func TestDeployBillingClose_Integration(t *testing.T) {
	reader := &fakeUsageReader{} //nolint:exhaustruct // zero value is an empty reader
	pusher := newFakePusher()
	closer := newFakeCloser()

	h := harness.New(t, harness.WithDeployBilling(reader, pusher, closer))

	// Two months ago: End() is always beyond the 24-hour ingestion window,
	// including when this test runs on the first day of a month.
	now := time.Now().UTC()
	closedPeriod := now.AddDate(0, -2, 0).Format("2006-01")
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
		reader.setActiveKeys([]clickhouse.ActiveKeysUsage{{WorkspaceID: wsID, ActiveKeys: 3}})
		invoiceID := uid.New("in")
		closer.setDrafts(subscriptionID, []invoicecloser.DraftInvoice{
			{ID: invoiceID, Status: "draft", BillingReason: "subscription_cycle", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
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
		require.Equal(t, int64(3), req.Values.ActiveKeys)

		require.Equal(t, p.End().Add(48*time.Hour).Unix(), closer.claimedAt(invoiceID))
		require.True(t, closer.didFinalize(invoiceID), "expected the ended cycle invoice to be finalized")
	})

	t.Run("old-period close neither claims nor finalizes next-period drafts", func(t *testing.T) {
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
			{ID: proration, Status: "draft", BillingReason: "subscription_update", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
			{ID: nextPeriod, Status: "draft", BillingReason: "subscription_cycle", PeriodStart: p.End().Unix(), PeriodEnd: p.End().AddDate(0, 1, 0).Unix()},
		})

		resp, err := runClose(closedPeriod, 0)
		require.NoError(t, err)
		require.Equal(t, int32(0), resp.GetInvoicesFinalized())
		require.Equal(t, int32(2), resp.GetInvoicesSkipped())
		require.Zero(t, closer.claimedAt(nextPeriod))
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
			{ID: invoiceID, Status: "draft", BillingReason: "subscription_cycle", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
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
			{ID: okInvoice, Status: "draft", BillingReason: "subscription_cycle", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
		})
		closer.setDrafts(failSub, []invoicecloser.DraftInvoice{
			{ID: failInvoice, Status: "draft", BillingReason: "subscription_cycle", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
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
	closedPeriod := now.AddDate(0, -2, 0).Format("2006-01")
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
		reader.setActiveKeys([]clickhouse.ActiveKeysUsage{{WorkspaceID: wsID, ActiveKeys: 3}})
		invoiceID := uid.New("in")
		closer.setDrafts(subscriptionID, []invoicecloser.DraftInvoice{
			{ID: invoiceID, Status: "draft", BillingReason: "subscription_cycle", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
		})

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
		require.Equal(t, int64(3), req.Values.ActiveKeys)
		require.True(t, closer.didFinalize(invoiceID))
	})

	t.Run("refuses to finalize a non-renewal invoice", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		workspaceID := uid.New(uid.WorkspacePrefix)
		invoiceID := uid.New("in")
		closer.setDrafts("sub_test", []invoicecloser.DraftInvoice{
			{ID: invoiceID, Status: "draft", BillingReason: "subscription_update", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
		})

		_, err := hydrav1.NewCronServiceIngressClient(h.Restate, workspaceID).
			CloseDeployBillingWorkspace().
			Request(h.Ctx, &hydrav1.CloseDeployBillingWorkspaceRequest{
				Period:    closedPeriod,
				PeriodEnd: 0,
				InvoiceId: invoiceID,
			})
		require.Error(t, err)
		require.False(t, closer.didFinalize(invoiceID))
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
			{ID: invoiceID, Status: "draft", BillingReason: "subscription_cycle", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
		})

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

	t.Run("already finalized invoice skips usage reads and meter pushes", func(t *testing.T) {
		reader.set(nil)
		pusher.reset()
		closer.reset()

		customerID := uid.New("cus")
		subscriptionID := uid.New("sub")
		wsID := seedBillableWorkspace(t, h, customerID, subscriptionID)
		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: wsID, ResourceID: "r1", CPUSeconds: 11},
		})
		invoiceID := uid.New("in")
		closer.setDrafts(subscriptionID, []invoicecloser.DraftInvoice{
			{ID: invoiceID, Status: "open", BillingReason: "subscription_cycle", PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix()},
		})

		_, err := hydrav1.NewCronServiceIngressClient(h.Restate, wsID).
			CloseDeployBillingWorkspace().
			Request(h.Ctx, &hydrav1.CloseDeployBillingWorkspaceRequest{
				Period:    closedPeriod,
				PeriodEnd: 0,
				InvoiceId: invoiceID,
			})
		require.NoError(t, err)

		instanceReads, activeKeyReads := reader.reads()
		require.Zero(t, instanceReads)
		require.Zero(t, activeKeyReads)
		_, pushed := pusher.get(customerID)
		require.False(t, pushed)
		require.False(t, closer.didFinalize(invoiceID))
	})
}
