package vercel

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Provider is an OpenFeature provider backed by a polled Vercel datafile.
// Evaluations read the last good snapshot and never block on the network.
// Initialization fetches the first datafile; [Provider.Run] keeps it fresh.
type Provider struct {
	config   Config
	endpoint string
	client   *http.Client

	snapshot atomic.Pointer[snapshot]
	health   health
}

// New validates config and returns a provider. Network access starts during
// OpenFeature initialization.
func New(config Config) (*Provider, error) {
	return newProvider(config, defaultEndpoint, &http.Client{})
}

func newProvider(config Config, endpoint string, client *http.Client) (*Provider, error) {
	config = config.withDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.New("endpoint must be an absolute HTTPS URL")
	}
	if err := assert.All(
		assert.Equal(parsed.Scheme, "https", "endpoint must be an absolute HTTPS URL"),
		assert.NotEmpty(parsed.Host, "endpoint must be an absolute HTTPS URL"),
		assert.True(parsed.User == nil, "endpoint must not contain credentials"),
	); err != nil {
		return nil, err
	}

	bounded := *client
	bounded.Timeout = config.HTTPTimeout
	bounded.CheckRedirect = rejectRedirect

	return &Provider{
		config:   config,
		endpoint: endpoint,
		client:   &bounded,
		snapshot: atomic.Pointer[snapshot]{},
		health:   health{mu: sync.Mutex{}, lastErr: "", failedAt: time.Time{}, failures: atomic.Uint64{}},
	}, nil
}

func rejectRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// Metadata identifies the provider to OpenFeature.
func (p *Provider) Metadata() openfeature.Metadata { return openfeature.Metadata{Name: "vercel"} }

// Hooks returns no provider hooks.
func (p *Provider) Hooks() []openfeature.Hook { return nil }

// InitWithContext fetches the first datafile. A failed fetch doesn't fail
// initialization: evaluations report PROVIDER_NOT_READY until [Provider.Run]
// refreshes successfully.
func (p *Provider) InitWithContext(ctx context.Context, _ openfeature.EvaluationContext) error {
	if err := p.refresh(ctx); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Warn("feature flag refresh failed, polling continues", "provider", "vercel", "error", err.Error())
	}
	return nil
}

// Init implements the context-free OpenFeature state handler.
func (p *Provider) Init(evaluationContext openfeature.EvaluationContext) error {
	return p.InitWithContext(context.Background(), evaluationContext)
}

// ShutdownWithContext does nothing: polling stops when the context passed to
// [Provider.Run] ends.
func (p *Provider) ShutdownWithContext(context.Context) error { return nil }

// Shutdown does nothing: polling stops when the context passed to
// [Provider.Run] ends.
func (p *Provider) Shutdown() {}

// Run refreshes the datafile every refresh interval until ctx ends. Failed
// refreshes keep the last good snapshot and are logged, so Run returns only
// when ctx ends, with nil. Start it with a runner:
//
//	r.Go(provider.Run)
func (p *Provider) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.config.RefreshInterval)
	defer ticker.Stop()

	failing := p.Diagnostics().LastRefreshError != ""
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		refreshCtx, cancel := context.WithTimeout(ctx, p.config.HTTPTimeout)
		err := p.refresh(refreshCtx)
		cancel()
		if ctx.Err() != nil {
			return nil
		}

		switch {
		case err != nil:
			logger.Warn("feature flag refresh failed", "provider", "vercel", "error", err.Error())
		case failing:
			logger.Info("feature flag refresh recovered", "provider", "vercel")
		}
		failing = err != nil
	}
}
