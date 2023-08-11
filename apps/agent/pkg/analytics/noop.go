package analytics

import "context"

type noop struct{}

func (n *noop) PublishKeyVerificationEvent(ctx context.Context, event KeyVerificationEvent) {}
func (n *noop) GetKeyStats(ctx context.Context, keyId string) (KeyStats, error) {
	return KeyStats{Usage: make([]ValueAtTime, 0)}, nil
}

func NewNoop() Analytics {
	return &noop{}
}
