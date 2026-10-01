package vercel

import "github.com/prometheus/client_golang/prometheus"

// Collectors returns the provider's refresh health metrics. They carry no
// labels, so they never expose flag, targeting, or key data.
func (p *Provider) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "feature_flags",
			Name:      "ready",
			Help:      "1 while the feature flag datafile is within its maximum staleness, else 0.",
		}, func() float64 {
			if p.Diagnostics().Ready {
				return 1
			}
			return 0
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "feature_flags",
			Name:      "snapshot_age_seconds",
			Help:      "Age of the feature flag datafile in use, or 0 before the first successful refresh.",
		}, func() float64 {
			return p.Diagnostics().SnapshotAge.Seconds()
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "feature_flags",
			Name:      "refresh_failures_total",
			Help:      "Failed feature flag datafile refreshes.",
		}, func() float64 {
			return float64(p.Diagnostics().RefreshFailures)
		}),
	}
}
