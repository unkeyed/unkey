//go:build linux

package prometheus

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	clientprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/require"
)

func TestSystemMetricsCollectorDependsOnPreviousGather(t *testing.T) {
	procRoot := t.TempDir()
	restoreHostProc(t, procRoot)

	writeCPUStat(t, procRoot, 100, 100)
	_, err := cpu.Percent(0, false)
	require.NoError(t, err)

	registry := clientprometheus.NewRegistry()
	registry.MustRegister(NewSystemMetricsCollector())

	writeCPUStat(t, procRoot, 140, 160)
	firstGather := gatherCPUPercent(t, registry)

	writeCPUStat(t, procRoot, 143, 160)
	secondGather := gatherCPUPercent(t, registry)

	require.InDelta(t, 40, firstGather, 0.001)
	require.InDelta(t, 100, secondGather, 0.001)
}

func restoreHostProc(t *testing.T, procRoot string) {
	t.Helper()

	original, wasSet := os.LookupEnv("HOST_PROC")
	require.NoError(t, os.Setenv("HOST_PROC", procRoot))
	t.Cleanup(func() {
		if wasSet {
			require.NoError(t, os.Setenv("HOST_PROC", original))
		} else {
			require.NoError(t, os.Unsetenv("HOST_PROC"))
		}

		_, err := cpu.Percent(0, false)
		require.NoError(t, err)
	})
}

func writeCPUStat(t *testing.T, procRoot string, busy, idle uint64) {
	t.Helper()

	stat := fmt.Appendf(nil, "cpu %d 0 0 %d 0 0 0 0 0 0\n", busy, idle)
	require.NoError(t, os.WriteFile(filepath.Join(procRoot, "stat"), stat, 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(procRoot, "meminfo"),
		[]byte("MemTotal: 1000 kB\nMemFree: 500 kB\nMemAvailable: 500 kB\n"),
		0o600,
	))
}

func gatherCPUPercent(t *testing.T, registry *clientprometheus.Registry) float64 {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != "resources_cpu_percent" {
			continue
		}

		require.Len(t, family.GetMetric(), 1)
		gauge := family.GetMetric()[0].GetGauge()
		require.NotNil(t, gauge)
		return gauge.GetValue()
	}

	t.Fatal("resources_cpu_percent was not collected")
	return 0
}
