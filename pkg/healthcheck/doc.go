// Package healthcheck provides push-based health monitoring for scheduled tasks.
//
// This package implements heartbeat clients for external monitoring services like
// Checkly. Heartbeats are sent after successful task completion to signal that
// scheduled jobs are running correctly.
//
// Basic usage:
//
//	// Create a Checkly heartbeat client
//	hb := healthcheck.NewChecklyHeartbeat("https://ping.checklyhq.com/abc123")
//
//	// Send heartbeat after successful task completion
//	if err := hb.Ping(ctx); err != nil {
//	    return fmt.Errorf("send heartbeat: %w", err)
//	}
//
//	// Use Noop for testing or when no heartbeat URL is configured
//	var hb healthcheck.Heartbeat = healthcheck.NewNoop()
//	if heartbeatURL != "" {
//	    hb = healthcheck.NewChecklyHeartbeat(heartbeatURL)
//	}
package healthcheck
