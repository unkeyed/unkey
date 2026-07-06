package enduserbilling

import "context"

// noopPusher discards pushes. Selected for identities bound to "export" or
// "none" (the export path is pull-based), and when no push provider is
// configured — the caller can still exercise its query/aggregation path
// without reporting usage.
type noopPusher struct{}

var _ MeterPusher = (*noopPusher)(nil)

// NewNoop returns a MeterPusher that reports nothing.
func NewNoop() MeterPusher { return &noopPusher{} }

func (n *noopPusher) Push(ctx context.Context, req PushRequest) (int, error) {
	return 0, nil
}
