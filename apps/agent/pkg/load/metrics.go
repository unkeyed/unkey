package load

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type hostMetrics struct {
	logger logging.Logger
	close  chan struct{}
}

type Config struct {
	Logger logging.Logger
}

func New(cfg Config) *hostMetrics {
	return &hostMetrics{
		close:  make(chan struct{}),
		logger: cfg.Logger,
	}
}

func (hm *hostMetrics) Start() {

	memoryUsed := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "host",
		Name:      "memory_used_bytes",
	})
	memoryMax := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "host",
		Name:      "memory_max_bytes",
	})
	memoryPercentage := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "host",
		Name:      "memory_usage_percent",
	})

	cpuPercentage := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "host",
		Name:      "cpu_usage_percent",
	})

	t := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-hm.close:
			t.Stop()
			return
		case <-t.C:
			v, err := mem.VirtualMemory()
			if err != nil {
				hm.logger.Error().Err(err).Msg("failed to get memory metrics")
				continue
			}

			cpuPercentages, err := cpu.Percent(0, false)
			if err != nil {
				hm.logger.Error().Err(err).Msg("failed to get cpu metrics")
				continue
			}
			cpuPercentage.Set(cpuPercentages[0])

			if len(cpuPercentages) != 1 {
				hm.logger.Error().Floats64("cpuPercentages", cpuPercentages).Msg("unexpected number of cpu percentages")
				continue
			}
			memoryUsed.Set(float64(v.Used))
			memoryMax.Set(float64(v.Total))
			memoryPercentage.Set(v.UsedPercent)

		}

	}

}

func (hm *hostMetrics) Stop() {
	hm.close <- struct{}{}
}
