// Package admin provides a client for the Restate admin API.
package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
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

// invocationIDPattern guards the IDs put into the introspection SQL query
// below. Restate invocation IDs are URL-safe tokens. The function rejects
// all other input instead of quoting it.
var invocationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// FindLiveInvocations reports which of the given invocation IDs still exist
// in Restate. It queries the admin SQL introspection endpoint (POST /query)
// against sys_invocation. That table only keeps rows for invocations that
// are pending, running, suspended, or retrying. A killed, purged, or
// completed invocation is not in it. The result maps every input ID to
// true when Restate knows it, and to false when it does not.
//
// The Accept header must be exactly "application/json". The endpoint
// compares the header value literally and answers with a binary Arrow
// stream for any other value.
func (c *Client) FindLiveInvocations(ctx context.Context, invocationIDs []string) (map[string]bool, error) {
	live := make(map[string]bool, len(invocationIDs))
	quoted := make([]string, 0, len(invocationIDs))
	for _, id := range invocationIDs {
		if !invocationIDPattern.MatchString(id) {
			return nil, fmt.Errorf("invalid invocation id %q", id)
		}
		live[id] = false
		quoted = append(quoted, "'"+id+"'")
	}
	if len(quoted) == 0 {
		return live, nil
	}

	payload, err := json.Marshal(map[string]string{
		"query": fmt.Sprintf("select id from sys_invocation where id in (%s)", strings.Join(quoted, ", ")),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/query", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("query failed with status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Rows []struct {
			ID string `json:"id"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}
	for _, row := range result.Rows {
		live[row.ID] = true
	}
	return live, nil
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
