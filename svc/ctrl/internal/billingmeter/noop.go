package billingmeter

import "context"

// noopPusher discards pushes. Used when the billing provider is not configured
// so the caller can still run (and exercise its query/aggregation path) without
// reporting usage. The per-workspace month-to-date numbers are logged by the
// caller, so a noop run still surfaces what we would have billed.
type noopPusher struct{}

var _ Pusher = (*noopPusher)(nil)

// NewNoop returns a Pusher that reports nothing.
func NewNoop() Pusher { return &noopPusher{} }

func (n *noopPusher) SetMonthToDate(ctx context.Context, req PushRequest) (int, error) {
	return 0, nil
}
