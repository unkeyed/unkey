package certificate

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) HealthMetricsHandler() http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(&healthCollector{
		db: s.db,
		failedChallenges: prometheus.NewDesc(
			"unkey_control_certificate_failed_challenges",
			"Current failed ACME challenges for existing domains, including initial issuance and renewal.",
			nil, nil,
		),
		earliestExpiry: prometheus.NewDesc(
			"unkey_control_certificate_earliest_expiry_timestamp_seconds",
			"Earliest known expiry of an issued certificate across all challenge states, or zero when none exist.",
			nil, nil,
		),
	})
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

type healthCollector struct {
	db               db.Database
	failedChallenges *prometheus.Desc
	earliestExpiry   *prometheus.Desc
}

func (c *healthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.failedChallenges
	ch <- c.earliestExpiry
}

func (c *healthCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	health, err := c.db.GetCertificateHealth(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.failedChallenges, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.failedChallenges, prometheus.GaugeValue, float64(health.FailedChallenges))
	ch <- prometheus.MustNewConstMetric(c.earliestExpiry, prometheus.GaugeValue, float64(health.EarliestExpiry)/1000)
}
