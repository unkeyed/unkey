package sdk

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/internal/sdk/internal/utils"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/internal/sdk/models/operations"
	"github.com/unkeyed/unkey/internal/sdk/polling"
)

// WaitForDeviceLogin uses generated polling with cancellable, server-paced waits.
// It retries transport, 429, and 5xx failures until ctx ends. A nil wait uses a timer.
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
	// Generated Go polling sleeps without observing context or Retry-After.
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
			if res.StatusCode == http.StatusAccepted {
				seconds, err := strconv.ParseInt(res.Header.Get("Retry-After"), 10, 64)
				if err == nil && seconds > 0 && seconds <= int64(math.MaxInt64/time.Second) {
					c.interval = time.Duration(seconds) * time.Second
				}
			}
			return res, nil
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
