package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/retry"
)

type restateRegistration struct {
	logger        logging.Logger
	adminURL      string
	registerAs    string
	acmeEnabled   bool
	dnsEnabled    bool
	defaultDomain string
	database      db.Database
	restateClient hydrav1.CertificateServiceIngressClient
}

func (r *restateRegistration) register(ctx context.Context) {
	// Wait a moment for the restate server to be ready
	time.Sleep(2 * time.Second)

	if err := r.doRegister(ctx); err != nil {
		r.logger.Error("failed to register with Restate after retries", "error", err)
		return
	}

	r.logger.Info("Successfully registered with Restate")
	r.bootstrapCertificates(ctx)
}

func (r *restateRegistration) doRegister(ctx context.Context) error {
	registerURL := fmt.Sprintf("%s/deployments", r.adminURL)
	payload := fmt.Sprintf(`{"uri": "%s"}`, r.registerAs)

	r.logger.Info("Registering with Restate", "admin_url", registerURL, "service_uri", r.registerAs)

	retrier := retry.New(
		retry.Attempts(10),
		retry.Backoff(func(n int) time.Duration {
			return 5 * time.Second
		}),
	)

	return retrier.Do(func() error {
		return r.sendRegistrationRequest(ctx, registerURL, payload)
	})
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

func (r *restateRegistration) bootstrapCertificates(ctx context.Context) {
	if !r.acmeEnabled || !r.dnsEnabled {
		return
	}

	if r.defaultDomain != "" {
		bootstrapWildcardDomain(ctx, r.database, r.logger, r.defaultDomain)
	}

	r.startCertRenewalCron(ctx)
}

func (r *restateRegistration) startCertRenewalCron(ctx context.Context) {
	_, err := r.restateClient.RenewExpiringCertificates().Send(
		ctx,
		&hydrav1.RenewExpiringCertificatesRequest{
			DaysBeforeExpiry: 30,
		},
		restate.WithIdempotencyKey("cert-renewal-cron-startup"),
	)
	if err != nil {
		r.logger.Warn("failed to start certificate renewal cron", "error", err)
		return
	}
	r.logger.Info("Certificate renewal cron job started")
}
