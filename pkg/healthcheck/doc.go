// Package healthcheck provides push-based health monitoring for scheduled tasks.
//
// Heartbeats are sent to external monitoring services after successful task
// completion to signal that scheduled jobs are running correctly.
//
// Basic usage:
//
//	// Create an HTTP heartbeat client
//	hb := healthcheck.NewHTTPHeartbeat("https://monitoring.example.com/heartbeat")
//
//	// Send heartbeat after successful task completion
//	if err := hb.Ping(ctx); err != nil {
//	    return fmt.Errorf("send heartbeat: %w", err)
//	}
//
//	// Use Noop for testing or when no heartbeat URL is configured
//	var hb healthcheck.Heartbeat = healthcheck.NewNoop()
//	if heartbeatURL != "" {
//	    hb = healthcheck.NewHTTPHeartbeat(heartbeatURL)
//	}
package healthcheck
