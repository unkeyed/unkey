package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/version"
)

// We're using const labels as workaround for the prometheus->otel adapter
// The adapter does not seem to export the resource lavels correctly and because
// it's temporary, we take the pragmatic approach here.
//
// Remove these after we've moved to pull based prometheus metrics.
var constLabels = prometheus.Labels{
	"region":  os.Getenv("UNKEY_REGION"),
	"version": version.Version,
}
