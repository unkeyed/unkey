package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/retry"
)

// restateRegistration handles self-registration with the Restate admin API.
// This is only needed when running outside of environments where registration
// is handled externally (e.g., local development).
type restateRegistration struct {
	logger     logging.Logger
	adminURL   string
	registerAs string
}

func (r *restateRegistration) register(ctx context.Context) {
	// Wait a moment for the restate server to be ready
	time.Sleep(2 * time.Second)

	registerURL := fmt.Sprintf("%s/deployments", r.adminURL)
	payload := fmt.Sprintf(`{"uri": "%s"}`, r.registerAs)

	r.logger.Info("Registering with Restate", "admin_url", registerURL, "service_uri", r.registerAs)

	retrier := retry.New(
		retry.Attempts(10),
		retry.Backoff(func(n int) time.Duration {
			return 5 * time.Second
		}),
	)

	err := retrier.Do(func() error {
		return r.sendRegistrationRequest(ctx, registerURL, payload)
	})
	if err != nil {
		r.logger.Error("failed to register with Restate after retries", "error", err)
		return
	}

	r.logger.Info("Successfully registered with Restate")
}

func (r *restateRegistration) sendRegistrationRequest(ctx context.Context, url, payload string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register with Restate: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	r.logger.Info("restate register response", "body", string(body))
	return fmt.Errorf("registration returned status %d", resp.StatusCode)
}
