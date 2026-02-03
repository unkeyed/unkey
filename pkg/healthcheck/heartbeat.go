package healthcheck

import "context"

// Heartbeat sends push-based health signals to monitoring services.
type Heartbeat interface {
	// Ping sends a heartbeat signal indicating successful task completion.
	// Returns an error if the heartbeat could not be sent.
	Ping(ctx context.Context) error
}
