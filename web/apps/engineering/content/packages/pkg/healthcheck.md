---
title: healthcheck
description: "provides push-based health monitoring for scheduled tasks"
---

Package healthcheck provides push-based health monitoring for scheduled tasks.

This package implements heartbeat clients for external monitoring services like Checkly. Heartbeats are sent after successful task completion to signal that scheduled jobs are running correctly.

Basic usage:

	// Create a Checkly heartbeat client
	hb := healthcheck.NewChecklyHeartbeat("https://ping.checklyhq.com/abc123")

	// Send heartbeat after successful task completion
	if err := hb.Ping(ctx); err != nil {
	    return fmt.Errorf("send heartbeat: %w", err)
	}

	// Use Noop for testing or when no heartbeat URL is configured
	var hb healthcheck.Heartbeat = healthcheck.NewNoop()
	if heartbeatURL != "" {
	    hb = healthcheck.NewChecklyHeartbeat(heartbeatURL)
	}

## Types

### type ChecklyHeartbeat

```go
type ChecklyHeartbeat struct {
	url    string
	client *http.Client
}
```

ChecklyHeartbeat sends heartbeats to Checkly monitoring service.

#### func NewChecklyHeartbeat

```go
func NewChecklyHeartbeat(url string) *ChecklyHeartbeat
```

NewChecklyHeartbeat creates a new Checkly heartbeat client.

#### func (ChecklyHeartbeat) Ping

```go
func (c *ChecklyHeartbeat) Ping(ctx context.Context) (err error)
```

Ping sends a heartbeat to Checkly.

### type Heartbeat

```go
type Heartbeat interface {
	// Ping sends a heartbeat signal indicating successful task completion.
	// Returns an error if the heartbeat could not be sent.
	Ping(ctx context.Context) error
}
```

Heartbeat sends push-based health signals to monitoring services.

### type Noop

```go
type Noop struct{}
```

Noop is a no-op heartbeat implementation for testing or when heartbeats are disabled.

#### func NewNoop

```go
func NewNoop() *Noop
```

NewNoop creates a new no-op heartbeat that does nothing.

#### func (Noop) Ping

```go
func (*Noop) Ping(context.Context) error
```

Ping does nothing and always returns nil.

