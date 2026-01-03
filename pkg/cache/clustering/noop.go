// nolint: all
package clustering

import (
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// noopDispatcher is a no-op implementation that does nothing.
// Used when clustering is disabled.
type noopDispatcher struct{}

func (n *noopDispatcher) Register(handler InvalidationHandler) {}
func (n *noopDispatcher) Close() error                         { return nil }

// NewNoopDispatcher creates a dispatcher that does nothing.
// Use this when clustering is disabled.
func NewNoopDispatcher() *InvalidationDispatcher {
	return &InvalidationDispatcher{
		handlers: make(map[string]InvalidationHandler),
		logger:   logging.NewNoop(),
	}
}
