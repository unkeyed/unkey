//go:build !linux

package collector

import (
	"context"
	"errors"
)

// criHandler matches the Linux build's interface so the collector code
// compiles across platforms.
type criHandler interface {
	OnStart(ctx context.Context, podUID string)
	OnExit(ctx context.Context, podUID string)
}

// criWatcher is unavailable off-Linux (containerd only runs there).
type criWatcher struct{}

func newCRIWatcher(_ string, _ criHandler) (*criWatcher, error) {
	return nil, errors.New("cri watcher not supported on this platform")
}

func (w *criWatcher) Run(_ context.Context) error { return nil }
func (w *criWatcher) Close() error                { return nil }
