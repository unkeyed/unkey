package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/unkeyed/unkey/pkg/logger"
)

// systemMetricsCollector exposes host-level CPU and memory utilization as
// Prometheus gauges. These mirror the resources_cpu_percent and
// resources_memory_percent metrics that previously rode the OTLP push
// pipeline; they now live on the Prometheus pull path alongside every other
// metric so we don't push what we already scrape.
type systemMetricsCollector struct {
	cpuPercent *prometheus.Desc
	memPercent *prometheus.Desc
}

// NewSystemMetricsCollector returns a Prometheus collector for host CPU and
// memory utilization percentages. Register it on the same registry that is
// exposed at /metrics.
func NewSystemMetricsCollector() prometheus.Collector {
	return &systemMetricsCollector{
		cpuPercent: prometheus.NewDesc(
			"resources_cpu_percent",
			"Host CPU utilization as a percentage (0-100).",
			nil, nil,
		),
		memPercent: prometheus.NewDesc(
			"resources_memory_percent",
			"Host memory utilization as a percentage (0-100).",
			nil, nil,
		),
	}
}

func (c *systemMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpuPercent
	ch <- c.memPercent
}

func (c *systemMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	// Interval 0 returns CPU usage since the previous call, which is
	// non-blocking and the idiomatic choice for scrape-driven collection
	// (a positive interval would block the scrape for that duration).
	if cpuPcts, err := cpu.Percent(0, false); err != nil {
		logger.Warn("failed to collect cpu percent", "error", err.Error())
	} else if len(cpuPcts) > 0 {
		ch <- prometheus.MustNewConstMetric(c.cpuPercent, prometheus.GaugeValue, cpuPcts[0])
	}

	if vm, err := mem.VirtualMemory(); err != nil {
		logger.Warn("failed to collect memory percent", "error", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(c.memPercent, prometheus.GaugeValue, vm.UsedPercent)
	}
}
