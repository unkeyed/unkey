// Package admin provides a client for the Restate admin API.
package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/retry"
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

type registrationPayload struct {
	URI   string `json:"uri"`
	Force bool   `json:"force,omitempty"`
}

type rule struct {
	Pattern      string       `json:"pattern"`
	Description  string       `json:"description"`
	Disabled     bool         `json:"disabled"`
	Limits       ruleLimits   `json:"limits"`
	Precondition precondition `json:"precondition"`
}

type ruleLimits struct {
	Concurrency int32 `json:"concurrency"`
}

type precondition struct {
	Type string `json:"type"`
}

func (c *Client) upsertRules(ctx context.Context, rules []rule) error {
	payload, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("marshal concurrency rules: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPut, c.baseURL+"/limits/rules", payload)
	if err != nil {
		return fmt.Errorf("upsert concurrency rules: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if readErr != nil {
		return fmt.Errorf("upsert concurrency rules failed with status %d (failed to read body: %w)", resp.StatusCode, readErr)
	}
	return fmt.Errorf("upsert concurrency rules failed with status %d: %s", resp.StatusCode, string(body))
}

// UpsertDefaultBuildConcurrencyRule ensures an unseen workspace is limited
// while its exact database-backed rule propagates to Restate's schedulers.
func (c *Client) UpsertDefaultBuildConcurrencyRule(ctx context.Context) error {
	return c.upsertRules(ctx, []rule{{
		Pattern:      "*",
		Description:  "Unkey default workspace build concurrency",
		Disabled:     false,
		Limits:       ruleLimits{Concurrency: 1},
		Precondition: precondition{Type: "none"},
	}})
}

// UpsertBuildConcurrencyRules configures the workspace and preview build
// concurrency rules as one atomic, unconditional update.
func (c *Client) UpsertBuildConcurrencyRules(ctx context.Context, workspaceID string, concurrency int32) error {
	previewConcurrency := concurrency - 1
	if previewConcurrency < 1 {
		previewConcurrency = 1
	}
	return c.upsertRules(ctx, []rule{
		{Pattern: workspaceID, Description: "Unkey workspace build concurrency", Disabled: false, Limits: ruleLimits{Concurrency: concurrency}, Precondition: precondition{Type: "none"}},
		{Pattern: workspaceID + "/preview", Description: "Unkey preview build concurrency", Disabled: false, Limits: ruleLimits{Concurrency: previewConcurrency}, Precondition: precondition{Type: "none"}},
	})
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
// If force is true, Restate will accept the deployment even if it detects
// incompatible changes (e.g. removed handlers).
// Retries up to 10 times with 5 second backoff on failure.
func (c *Client) RegisterDeployment(ctx context.Context, uri string, force ...bool) error {
	retrier := retry.New(
		retry.Attempts(10),
		retry.Backoff(func(n int) time.Duration {
			return 5 * time.Second
		}),
	)

	return retrier.Do(func() error {
		return c.registerDeployment(ctx, uri, len(force) > 0 && force[0])
	})
}

func (c *Client) registerDeployment(ctx context.Context, uri string, force bool) error {
	url := fmt.Sprintf("%s/deployments", c.baseURL)

	payload, err := json.Marshal(registrationPayload{URI: uri, Force: force})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("registration failed with status %d (failed to read body: %w)", resp.StatusCode, readErr)
	}
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

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("cancel failed with status %d (failed to read body: %w)", resp.StatusCode, readErr)
	}
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
