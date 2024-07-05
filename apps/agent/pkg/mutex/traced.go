package mutex

import (
	"context"
	"sync"

	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

// Lock is a wrapper around sync.RWMutex that traces lock and unlock operations.
type Lock struct {
	mu sync.RWMutex
}

func New() *Lock {
	return &Lock{
		mu: sync.RWMutex{},
	}
}

func (m *Lock) Lock(ctx context.Context) {
	_, span := tracing.Start(ctx, "Lock")
	defer span.End()
	m.mu.Lock()
}

func (m *Lock) RLock(ctx context.Context) {
	_, span := tracing.Start(ctx, "RLock")
	defer span.End()
	m.mu.RLock()
}

func (m *Lock) Unlock(ctx context.Context) {
	_, span := tracing.Start(ctx, "RUnlock")
	defer span.End()
	m.mu.Unlock()
}

func (m *Lock) RUnlock(ctx context.Context) {
	_, span := tracing.Start(ctx, "RUnlock")
	defer span.End()
	m.mu.RUnlock()
}
