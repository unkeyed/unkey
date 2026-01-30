package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ChecklyHeartbeat sends heartbeats to Checkly monitoring service.
type ChecklyHeartbeat struct {
	url    string
	client *http.Client
}

// NewChecklyHeartbeat creates a new Checkly heartbeat client.
func NewChecklyHeartbeat(url string) *ChecklyHeartbeat {
	return &ChecklyHeartbeat{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Ping sends a heartbeat to Checkly.
func (c *ChecklyHeartbeat) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("heartbeat failed: status %d", resp.StatusCode)
	}

	return nil
}
