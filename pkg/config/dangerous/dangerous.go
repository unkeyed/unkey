// Package dangerous keeps security bypasses outside normal service configuration.
package dangerous

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

const envBypassGvisor = "DANGEROUSLY_BYPASS_GVISOR_AND_GET_US_PWN3D"

var enabled = vec.NewGauge(prometheus.GaugeOpts{
	Namespace: "unkey",
	Name:      "dangerous_flag_enabled",
	Help:      "Whether a dangerous flag is enabled (1) or disabled (0).",
}, vec.Values("flag", envBypassGvisor))

// DangerouslyByPassGVisor requires the exact environment value "true" to disable isolation.
// Call once at startup so the warning and gauge describe the running service.
func DangerouslyByPassGVisor() bool {
	if os.Getenv(envBypassGvisor) == "true" {
		logger.Warn("gVisor sandbox disabled; user workloads can access the host kernel", "flag", envBypassGvisor)
		enabled.WithLabelValues(envBypassGvisor).Set(1)
		return true
	}
	enabled.WithLabelValues(envBypassGvisor).Set(0)
	return false
}
