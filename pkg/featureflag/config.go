package featureflag

import (
	"context"
	"fmt"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/featureflag/static"
	"github.com/unkeyed/unkey/pkg/featureflag/vercel"
	"github.com/unkeyed/unkey/pkg/runner"
)

// Provider names the backend a service reads feature flags from.
type Provider string

const (
	// ProviderNone resolves every flag to its code default without network
	// access. It is the zero value, so unconfigured services keep gated
	// features off.
	ProviderNone Provider = ""

	// ProviderStatic resolves flags from [Config.Static]. For local
	// development and tests only.
	ProviderStatic Provider = "static"

	// ProviderVercel polls the Vercel Flags datafile. See
	// [github.com/unkeyed/unkey/pkg/featureflag/vercel] for the supported subset.
	ProviderVercel Provider = "vercel"
)

const initTimeout = time.Minute

// Config selects and configures a feature flag backend.
type Config struct {
	Provider Provider `toml:"provider"`

	Static static.Values  `toml:"static"`
	Vercel *vercel.Config `toml:"vercel"`
}

// Validate checks that the selected backend has the settings it needs.
func (c Config) Validate() error {
	switch c.Provider {
	case ProviderNone:
		return nil
	case ProviderStatic:
		return c.Static.Validate()
	case ProviderVercel:
		return assert.True(c.Vercel != nil, "feature_flags.vercel is required when the vercel provider is selected")
	default:
		return fmt.Errorf("invalid feature flag provider %q: must be empty, %q, or %q", c.Provider, ProviderStatic, ProviderVercel)
	}
}

// NewFromConfig builds an isolated OpenFeature API for the configured backend.
// It registers the backend's health metrics on reg, starts its background
// refresh on r, and shuts the API down with r. The Vercel backend starts even
// when its first refresh fails; evaluations report PROVIDER_NOT_READY until a
// later refresh succeeds.
func NewFromConfig(ctx context.Context, r *runner.Runner, cfg Config, reg prometheus.Registerer) (*openfeature.EvaluationAPI, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var provider openfeature.FeatureProvider
	var refresh runner.RunFunc
	switch cfg.Provider {
	case ProviderNone:
	case ProviderStatic:
		provider = static.New(cfg.Static)
	case ProviderVercel:
		vercelProvider, err := vercel.New(*cfg.Vercel)
		if err != nil {
			return nil, fmt.Errorf("configure vercel feature flags: %w", err)
		}
		for _, collector := range vercelProvider.Collectors() {
			if err := reg.Register(collector); err != nil {
				return nil, fmt.Errorf("register feature flag metrics: %w", err)
			}
		}
		provider, refresh = vercelProvider, vercelProvider.Run
	}

	initCtx, cancel := context.WithTimeout(ctx, initTimeout)
	defer cancel()
	api, err := New(initCtx, provider)
	if err != nil {
		return nil, err
	}
	r.DeferCtx(api.Shutdown)
	if refresh != nil {
		r.Go(refresh)
	}
	return api, nil
}
