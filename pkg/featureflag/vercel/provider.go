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
type Provider struct {
	config   Config
	endpoint string
	client   *http.Client

	snapshot atomic.Pointer[snapshot]
	health   health

	lifecycle sync.Mutex
	stop      context.CancelFunc
	stopped   chan struct{}
}

// New validates config and returns a provider. Network access starts during
// OpenFeature initialization.
func New(config Config) (*Provider, error) {
	return newProvider(config, defaultEndpoint, &http.Client{})
}

func newProvider(config Config, endpoint string, client *http.Client) (*Provider, error) {
	config = config.withDefaults()
	if err := config.validate(); err != nil {
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
		config:    config,
		endpoint:  endpoint,
		client:    &bounded,
		snapshot:  atomic.Pointer[snapshot]{},
		health:    health{mu: sync.Mutex{}, lastErr: "", failedAt: time.Time{}, failures: atomic.Uint64{}},
		lifecycle: sync.Mutex{},
		stop:      nil,
		stopped:   nil,
	}, nil
}

func rejectRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// Metadata identifies the provider to OpenFeature.
func (p *Provider) Metadata() openfeature.Metadata { return openfeature.Metadata{Name: "vercel"} }

// Hooks returns no provider hooks.
func (p *Provider) Hooks() []openfeature.Hook { return nil }

// InitWithContext fetches the first datafile and starts polling. A failed
// first fetch doesn't fail initialization: evaluations report
// PROVIDER_NOT_READY until a later poll succeeds.
func (p *Provider) InitWithContext(ctx context.Context, _ openfeature.EvaluationContext) error {
	p.lifecycle.Lock()
	defer p.lifecycle.Unlock()
	if p.stop != nil {
		return errors.New("vercel feature flags provider already initialized")
	}

	if err := p.refresh(ctx); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Warn("feature flag refresh failed, polling continues", "provider", "vercel", "error", err.Error())
	}

	pollCtx, stop := context.WithCancel(context.WithoutCancel(ctx))
	p.stop = stop
	p.stopped = make(chan struct{})
	go p.poll(pollCtx, p.stopped)
	return nil
}

// Init implements the context-free OpenFeature state handler.
func (p *Provider) Init(evaluationContext openfeature.EvaluationContext) error {
	return p.InitWithContext(context.Background(), evaluationContext)
}

func (p *Provider) poll(ctx context.Context, stopped chan<- struct{}) {
	defer close(stopped)
	ticker := time.NewTicker(p.config.RefreshInterval)
	defer ticker.Stop()

	failing := p.Diagnostics().LastRefreshError != ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		refreshCtx, cancel := context.WithTimeout(ctx, p.config.HTTPTimeout)
		err := p.refresh(refreshCtx)
		cancel()
		if ctx.Err() != nil {
			return
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

// ShutdownWithContext stops polling and waits for the poller to exit.
func (p *Provider) ShutdownWithContext(ctx context.Context) error {
	p.lifecycle.Lock()
	defer p.lifecycle.Unlock()
	if p.stop == nil {
		return nil
	}

	p.stop()
	select {
	case <-p.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown implements the context-free OpenFeature state handler.
func (p *Provider) Shutdown() {
	p.lifecycle.Lock()
	defer p.lifecycle.Unlock()
	if p.stop == nil {
		return
	}
	p.stop()
	<-p.stopped
}
