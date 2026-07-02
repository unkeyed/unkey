package billingmeter

import "context"

// Pusher reports one workspace's month-to-date meter totals to the billing
// provider. Implementations set (not increment) the period quantity, so
// repeated calls with the same or a newer total are idempotent. Returns the
// number of meter events pushed.
type Pusher interface {
	Push(ctx context.Context, req PushRequest) (int, error)
}

// PushRequest is one workspace's month-to-date usage to report. The provider
// maps usage to a customer by StripeCustomerID, so the customer ID (not a
// subscription or price) is all the caller needs to supply.
type PushRequest struct {
	StripeCustomerID string
	Values           MeterValues
	// Timestamp is the unix-seconds instant recorded on the meter event. With
	// "last" aggregation the event with the newest timestamp in the period
	// wins, so this is what makes the latest hourly push the billed value.
	Timestamp int64
}

// MeterValues is the month-to-date usage for one workspace, in the exact units
// each meter expects. These are sent as decimal strings, so they are kept as
// floats and never rounded.
type MeterValues struct {
	// CPUSeconds is CPU time used, in CPU-seconds.
	CPUSeconds float64
	// MemoryGiBSeconds is memory integrated over time, in GiB-seconds.
	MemoryGiBSeconds float64
	// EgressGiB is public egress, in binary GiB.
	EgressGiB float64
	// DiskGiBSeconds is allocated disk integrated over time, in GiB-seconds.
	DiskGiBSeconds float64
	// ActiveKeys is the number of distinct keys verified through the Deploy
	// gateway this period (month-to-date, like every other meter).
	ActiveKeys float64
}

// Positive reports whether any meter has usage worth pushing.
func (v MeterValues) Positive() bool {
	return v.CPUSeconds > 0 || v.MemoryGiBSeconds > 0 || v.EgressGiB > 0 || v.DiskGiBSeconds > 0 ||
		v.ActiveKeys > 0
}
