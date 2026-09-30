package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/featureflag"
	"github.com/unkeyed/unkey/pkg/featureflag/vercel"
)

const featureFlagInitTimeout = time.Minute

func newFeatureFlags(ctx context.Context, cfg FeatureFlagsConfig, reg promclient.Registerer) (*openfeature.EvaluationAPI, error) {
	var provider openfeature.FeatureProvider
	switch cfg.Provider {
	case FeatureFlagProviderNone:
	case FeatureFlagProviderStatic:
		provider = featureflag.Static(cfg.Static)
	case FeatureFlagProviderVercel:
		vercelProvider, err := vercel.New(vercel.Config{
			SDKKey:          cfg.Vercel.SDKKey,
			RefreshInterval: cfg.Vercel.RefreshInterval,
			HTTPTimeout:     cfg.Vercel.HTTPTimeout,
			MaxStaleness:    cfg.Vercel.MaxStaleness,
		})
		if err != nil {
			return nil, fmt.Errorf("configure vercel feature flags: %w", err)
		}
		if err := registerVercelFlagMetrics(reg, vercelProvider); err != nil {
			return nil, err
		}
		provider = vercelProvider
	default:
		return nil, fmt.Errorf("invalid feature flag provider %q", cfg.Provider)
	}

	initCtx, cancel := context.WithTimeout(ctx, featureFlagInitTimeout)
	defer cancel()
	return featureflag.New(initCtx, provider)
}

func registerVercelFlagMetrics(reg promclient.Registerer, provider *vercel.Provider) error {
	collectors := []promclient.Collector{
		promclient.NewGaugeFunc(promclient.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "feature_flags_ready",
			Help:      "1 while the feature flag datafile is within its maximum staleness, else 0.",
		}, func() float64 {
			if provider.Diagnostics().Ready {
				return 1
			}
			return 0
		}),
		promclient.NewGaugeFunc(promclient.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "feature_flags_snapshot_age_seconds",
			Help:      "Age of the feature flag datafile in use, or 0 before the first successful refresh.",
		}, func() float64 {
			return provider.Diagnostics().SnapshotAge.Seconds()
		}),
		promclient.NewCounterFunc(promclient.CounterOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "feature_flags_refresh_failures_total",
			Help:      "Failed feature flag datafile refreshes.",
		}, func() float64 {
			return float64(provider.Diagnostics().RefreshFailures)
		}),
	}
	for _, collector := range collectors {
		if err := reg.Register(collector); err != nil {
			return fmt.Errorf("register feature flag metrics: %w", err)
		}
	}
	return nil
}
