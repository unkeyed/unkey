package network

import "k8s.io/apimachinery/pkg/types"

type noop struct{}

// NewNoop returns a Reader that measures nothing. Use it when network
// collection is disabled so callers can use the Reader unconditionally.
// Read always returns ErrNotAttached, so every checkpoint stays unmeasured
// rather than reporting zero bytes. MapEntries reports an empty map.
func NewNoop() Reader { return noop{} }

func (noop) Attach(types.UID) error           { return nil }
func (noop) Detach(types.UID)                 {}
func (noop) Read(types.UID) (Counters, error) { return zeroCounters, ErrNotAttached }
func (noop) MapEntries() (int, error)         { return 0, nil }
func (noop) Reconcile(map[types.UID]struct{}) {}
func (noop) Close() error                     { return nil }
