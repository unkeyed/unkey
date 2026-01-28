// Package admin provides a client for the Restate admin API.
package admin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides access to the Restate admin API for managing deployments
// and invocations.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Config holds configuration for creating a new admin [Client].
type Config struct {
	// BaseURL is the Restate admin API endpoint (e.g., "http://restate:9070").
	BaseURL string
	// APIKey is the optional authentication key for admin API requests.
	APIKey string
}

// New creates a new admin [Client] with the given configuration.
func New(cfg Config) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterDeployment registers a service deployment with Restate.
// The uri should be the endpoint where Restate can reach the service.
func (c *Client) RegisterDeployment(ctx context.Context, uri string) error {
	url := fmt.Sprintf("%s/deployments", c.baseURL)
	payload := fmt.Sprintf(`{"uri": "%s"}`, uri)

	resp, err := c.do(ctx, http.MethodPost, url, []byte(payload))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
}

// CancelInvocation cancels a running invocation.
// Returns nil if the invocation was successfully canceled or if it was not found
// (already completed or never existed).
func (c *Client) CancelInvocation(ctx context.Context, invocationID string) error {
	url := fmt.Sprintf("%s/invocations/%s/cancel", c.baseURL, invocationID)

	resp, err := c.do(ctx, http.MethodPatch, url, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// 202 Accepted = cancellation initiated
	// 404 Not Found = invocation already completed or never existed
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("cancel failed with status %d: %s", resp.StatusCode, string(body))
}

func (c *Client) do(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}
