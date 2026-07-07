package stripe

import (
	"context"
	"fmt"
	"time"

	stripesdk "github.com/stripe/stripe-go/v86"
)

// ClockRow is one customer row under a test clock for list UIs.
type ClockRow struct {
	ClockID         string
	ClockName       string
	Status          stripesdk.TestHelpersTestClockStatus
	FrozenTime      int64
	CustomerID      string
	WorkspaceID     string
	PeriodEnd       int64
	HasSubscription bool
}

// ListClockRows returns flattened clock/customer rows for interactive UIs.
func ListClockRows(ctx context.Context, sc *stripesdk.Client) ([]ClockRow, error) {
	var rows []ClockRow
	clocks := sc.V1TestHelpersTestClocks.List(ctx, &stripesdk.TestHelpersTestClockListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(20)},
	})
	for clock, err := range clocks.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("list test clocks: %w", err)
		}

		customers, err := listClockCustomers(ctx, sc, clock.ID)
		if err != nil {
			return nil, err
		}
		if len(customers) == 0 {
			rows = append(rows, ClockRow{
				ClockID:         clock.ID,
				ClockName:       clock.Name,
				Status:          clock.Status,
				FrozenTime:      clock.FrozenTime,
				CustomerID:      "",
				WorkspaceID:     "",
				PeriodEnd:       0,
				HasSubscription: false,
			})
			continue
		}
		for _, customer := range customers {
			end, hasSub, err := latestPeriodEnd(ctx, sc, customer.ID)
			if err != nil {
				return nil, err
			}
			rows = append(rows, ClockRow{
				ClockID:         clock.ID,
				ClockName:       clock.Name,
				Status:          clock.Status,
				FrozenTime:      clock.FrozenTime,
				CustomerID:      customer.ID,
				WorkspaceID:     customer.Metadata["workspace_id"],
				PeriodEnd:       end,
				HasSubscription: hasSub,
			})
		}
	}
	return rows, nil
}

// AdvanceOptions configures how a test clock is advanced.
type AdvanceOptions struct {
	ToRFC3339 string
	Hours     float64
}

// AdvanceProgress reports asynchronous clock advancement status.
type AdvanceProgress struct {
	Status string
	Done   bool
	Err    error
	Frozen int64
}

// AdvanceClock advances a test clock and polls until ready.
func AdvanceClock(ctx context.Context, sc *stripesdk.Client, clockID string, opts AdvanceOptions, onProgress func(AdvanceProgress)) error {
	clock, err := sc.V1TestHelpersTestClocks.Retrieve(ctx, clockID, nil)
	if err != nil {
		return fmt.Errorf("retrieve clock %s: %w", clockID, err)
	}

	target, err := ResolveTargetTime(ctx, sc, clock, opts)
	if err != nil {
		return err
	}
	if target <= clock.FrozenTime {
		return fmt.Errorf("target %s is not after the clock's %s", FormatTime(target), FormatTime(clock.FrozenTime))
	}

	_, err = sc.V1TestHelpersTestClocks.Advance(ctx, clockID, &stripesdk.TestHelpersTestClockAdvanceParams{
		FrozenTime: stripesdk.Int64(target),
	})
	if err != nil {
		return fmt.Errorf("advance clock: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		time.Sleep(2 * time.Second)
		current, err := sc.V1TestHelpersTestClocks.Retrieve(ctx, clockID, nil)
		if err != nil {
			return fmt.Errorf("poll clock: %w", err)
		}
		if current.Status == stripesdk.TestHelpersTestClockStatusReady {
			if onProgress != nil {
				onProgress(AdvanceProgress{
					Status: string(current.Status),
					Done:   true,
					Err:    nil,
					Frozen: current.FrozenTime,
				})
			}
			return nil
		}
		if current.Status == stripesdk.TestHelpersTestClockStatusInternalFailure {
			return fmt.Errorf("stripe reported an internal failure advancing the clock")
		}
		if onProgress != nil {
			onProgress(AdvanceProgress{
				Status: string(current.Status),
				Done:   false,
				Err:    nil,
				Frozen: 0,
			})
		}
	}
}

// DeleteClock removes a test clock and its customers.
func DeleteClock(ctx context.Context, sc *stripesdk.Client, clockID string) ([]string, error) {
	customers, err := listClockCustomers(ctx, sc, clockID)
	if err != nil {
		return nil, err
	}
	if _, err := sc.V1TestHelpersTestClocks.Delete(ctx, clockID, nil); err != nil {
		return nil, fmt.Errorf("delete clock %s: %w", clockID, err)
	}
	ids := make([]string, 0, len(customers))
	for _, customer := range customers {
		ids = append(ids, customer.ID)
	}
	return ids, nil
}

// ResolveClockID returns a clock id from either clock or customer id.
func ResolveClockID(ctx context.Context, sc *stripesdk.Client, clockID, customerID string) (string, error) {
	if clockID != "" {
		return clockID, nil
	}
	customer, err := sc.V1Customers.Retrieve(ctx, customerID, nil)
	if err != nil {
		return "", fmt.Errorf("retrieve customer %s: %w", customerID, err)
	}
	if customer.TestClock == nil {
		return "", fmt.Errorf("customer %s is not on a test clock", customerID)
	}
	return customer.TestClock.ID, nil
}

// ResolveTargetTime picks the unix timestamp a clock advance will move to.
func ResolveTargetTime(ctx context.Context, sc *stripesdk.Client, clock *stripesdk.TestHelpersTestClock, opts AdvanceOptions) (int64, error) {
	if opts.ToRFC3339 != "" {
		t, err := time.Parse(time.RFC3339, opts.ToRFC3339)
		if err != nil {
			return 0, fmt.Errorf("parse target time: %w", err)
		}
		return t.Unix(), nil
	}
	if opts.Hours > 0 {
		return clock.FrozenTime + int64(opts.Hours*3600), nil
	}

	customers, err := listClockCustomers(ctx, sc, clock.ID)
	if err != nil {
		return 0, err
	}
	var latest int64
	for _, customer := range customers {
		end, ok, err := latestPeriodEnd(ctx, sc, customer.ID)
		if err != nil {
			return 0, err
		}
		if ok && end > latest {
			latest = end
		}
	}
	if latest == 0 {
		return 0, fmt.Errorf("no subscriptions on this clock; pass hours or an absolute time instead")
	}
	return latest + 2*3600, nil
}

func listClockCustomers(ctx context.Context, sc *stripesdk.Client, clockID string) ([]*stripesdk.Customer, error) {
	list := sc.V1Customers.List(ctx, &stripesdk.CustomerListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(5)},
		TestClock:  stripesdk.String(clockID),
	})
	var customers []*stripesdk.Customer
	for customer, err := range list.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("list customers on clock %s: %w", clockID, err)
		}
		customers = append(customers, customer)
	}
	return customers, nil
}

func latestPeriodEnd(ctx context.Context, sc *stripesdk.Client, customerID string) (int64, bool, error) {
	list := sc.V1Subscriptions.List(ctx, &stripesdk.SubscriptionListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(10)},
		Customer:   stripesdk.String(customerID),
	})
	var latest int64
	found := false
	for sub, err := range list.All(ctx) {
		if err != nil {
			return 0, false, fmt.Errorf("list subscriptions for %s: %w", customerID, err)
		}
		for _, item := range sub.Items.Data {
			found = true
			if item.CurrentPeriodEnd > latest {
				latest = item.CurrentPeriodEnd
			}
		}
	}
	return latest, found, nil
}
