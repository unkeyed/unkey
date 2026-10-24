package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/sdk/internal/utils"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/internal/sdk/models/operations"
	"github.com/unkeyed/unkey/internal/sdk/polling"
)

// WaitForDeviceLogin uses generated polling with cancellable, server-paced waits.
// It retries transport, 429, and 5xx failures until ctx ends. A nil wait uses a timer.
// Denied or expired logins return both the response and a polling.FailureCriteriaError.
func (c *Client) WaitForDeviceLogin(ctx context.Context, request components.V2CliExchangeTokenRequestBody, interval time.Duration, wait func(context.Context, time.Duration) error) (*operations.CliExchangeTokenResponse, error) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if wait == nil {
		wait = waitContext
	}
	api := New(
		WithServerURL(utils.ReplaceParameters(c.sdkConfiguration.GetServerDetails())),
		WithClient(&deviceLoginClient{client: c.sdkConfiguration.Client, interval: interval, wait: wait, started: false}),
	)
	// Generated Go polling sleeps without observing context or response intervals.
	// The transport supplies those waits; generated code owns terminal criteria.
	res, err := api.Cli.ExchangeToken(ctx, request, operations.WithPolling(
		api.Cli.ExchangeTokenWaitForCompletion(), polling.WithIntervalSecondsOverride(0),
	))
	var response *ResponseError
	if errors.As(err, &response) {
		return res, response
	}
	return res, err
}

type deviceLoginClient struct {
	client   HTTPClient
	interval time.Duration
	wait     func(context.Context, time.Duration) error
	started  bool
}

func (c *deviceLoginClient) Do(req *http.Request) (*http.Response, error) {
	for {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		if c.started {
			if err := c.wait(req.Context(), c.interval); err != nil {
				return nil, err
			}
		}
		c.started = true
		res, err := c.client.Do(req)
		if err == nil {
			return c.updateInterval(res)
		}
		var transport *TransportError
		var response *ResponseError
		if !errors.As(err, &transport) && !(errors.As(err, &response) && (response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500)) {
			return nil, err
		}
		if req.GetBody == nil {
			return nil, err
		}
		req.Body, err = req.GetBody()
		if err != nil {
			return nil, err
		}
	}
}

func (c *deviceLoginClient) updateInterval(res *http.Response) (*http.Response, error) {
	payload, readErr := io.ReadAll(res.Body)
	closeErr := res.Body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	res.Body = io.NopCloser(bytes.NewReader(payload))
	if res.StatusCode == http.StatusOK {
		var body components.V2CliExchangeTokenResponseBody
		if err := json.Unmarshal(payload, &body); err != nil {
			return nil, err
		}
		if !body.Data.Status.IsExact() {
			return nil, &ResponseError{StatusCode: res.StatusCode, Detail: fmt.Sprintf("unexpected login status %q", body.Data.Status)}
		}
		if body.Data.Interval > 0 {
			c.interval = time.Duration(body.Data.Interval) * time.Second
		}
	}
	return res, nil
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
