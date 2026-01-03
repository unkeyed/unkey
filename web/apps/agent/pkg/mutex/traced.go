package mutex

import (
	"context"
	"sync"

	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

// Lock is a wrapper around sync.RWMutex that traces lock and unlock operations.
type TraceLock struct {
	mu sync.RWMutex
}

func New() *TraceLock {
	return &TraceLock{
		mu: sync.RWMutex{},
	}
}

func (l *TraceLock) Lock(ctx context.Context) {
	_, span := tracing.Start(ctx, "Lock")
	defer span.End()
	l.mu.Lock()
}

func (l *TraceLock) RLock(ctx context.Context) {
	_, span := tracing.Start(ctx, "RLock")
	defer span.End()
	l.mu.RLock()
}

func (l *TraceLock) Unlock(ctx context.Context) {
	_, span := tracing.Start(ctx, "RUnlock")
	defer span.End()
	l.mu.Unlock()
}

func (l *TraceLock) RUnlock(ctx context.Context) {
	_, span := tracing.Start(ctx, "RUnlock")
	defer span.End()
	l.mu.RUnlock()
}
