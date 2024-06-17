package load

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
)

type hostMetrics struct {
	logger  logging.Logger
	metrics metrics.Metrics
	close   chan struct{}
}

type Config struct {
	Logger  logging.Logger
	Metrics metrics.Metrics
}

func New(cfg Config) *hostMetrics {
	return &hostMetrics{
		close:   make(chan struct{}),
		logger:  cfg.Logger,
		metrics: cfg.Metrics,
	}
}

func (hm *hostMetrics) Start() {
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
			if len(cpuPercentages) != 1 {
				hm.logger.Error().Floats64("cpuPercentages", cpuPercentages).Msg("unexpected number of cpu percentages")
				continue
			}

			m := metrics.SystemLoadReport{}
			m.CpuUsage = cpuPercentages[0]
			m.Memory.Total = v.Total
			m.Memory.Used = v.Used
			m.Memory.Percentage = v.UsedPercent
			hm.logger.Debug().Msgf("system load - CPU: %.2f%%, Memory: %.2f%%, MemoryTotal: %.2f GB", m.CpuUsage, m.Memory.Percentage, float64(m.Memory.Total)/1024/1024/1024)
			hm.metrics.ReportSystemLoad(m)

		}

	}

}

func (hm *hostMetrics) Stop() {
	hm.close <- struct{}{}
}
