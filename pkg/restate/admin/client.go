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
	return errorFromResponse("registration", resp)
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
	return errorFromResponse("cancel", resp)
}

// invocationIDPattern guards the IDs put into the introspection SQL query
// below. The admin API /query endpoint takes one SQL string and has no
// bind parameters, so the IDs must be quoted into the query text. The
// pattern rejects everything that could escape a quoted string.
var invocationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// FindLiveInvocations reports which of the given invocation IDs are still
// executing in Restate. It queries the admin SQL introspection endpoint
// (POST /query) against sys_invocation. The result maps every input ID to
// true when its invocation is live (pending, running, suspended, or in
// retry), and to false when it is not.
//
// Two behaviors of the endpoint shape the query, both verified against
// Restate 1.6.0:
//
//   - A direct comparison on the id column makes Restate parse each
//     literal as an invocation ID and the whole query fails with a 500
//     when one does not parse. concat(id, ”) forces a plain string
//     comparison, so unknown or malformed IDs return no row instead of
//     an error.
//   - A killed or cancelled invocation keeps a sys_invocation row with
//     status 'completed' until its retention expires or it is purged.
//     The status filter excludes those rows; without it a killed
//     invocation counts as live for hours.
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
		"query": fmt.Sprintf(
			"select id from sys_invocation where concat(id, '') in (%s) and status <> 'completed'",
			strings.Join(quoted, ", "),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, c.baseURL+"/query", payload)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errorFromResponse("query", resp)
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

// errorFromResponse turns a non-success admin API response into an error
// that includes the response body. The caller still owns the body close.
func errorFromResponse(action string, resp *http.Response) error {
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("%s failed with status %d (failed to read body: %w)", action, resp.StatusCode, readErr)
	}
	return fmt.Errorf("%s failed with status %d: %s", action, resp.StatusCode, string(body))
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

	// The value must be exactly "application/json". The /query endpoint
	// compares it literally and answers with a binary Arrow stream for
	// any other value.
	req.Header.Set("Accept", "application/json")

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}
