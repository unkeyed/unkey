package dangerous_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config/dangerous"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
)

func TestDangerouslyByPassGVisor_WarnsAndExportsEnabledState(t *testing.T) {
	t.Setenv("DANGEROUSLY_BYPASS_GVISOR_AND_GET_US_PWN3D", "true")
	logs := loggertest.Install(t)
	require.True(t, dangerous.DangerouslyByPassGVisor())
	record := logs.Last(t)
	require.Equal(t, slog.LevelWarn, record.Level)
	require.Equal(t, "DANGEROUSLY_BYPASS_GVISOR_AND_GET_US_PWN3D", loggertest.FlatAttrs(record)["flag"])
	requireGauge(t, 1)
}

func TestDangerouslyByPassGVisor_RequiresExplicitTrue(t *testing.T) {
	for _, value := range []string{"unset", "", "false", "1", "t", "T", "TRUE", "True", " true ", "typo"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("DANGEROUSLY_BYPASS_GVISOR_AND_GET_US_PWN3D", value)
			if value == "unset" {
				require.NoError(t, os.Unsetenv("DANGEROUSLY_BYPASS_GVISOR_AND_GET_US_PWN3D"))
			}
			logs := loggertest.Install(t)
			require.False(t, dangerous.DangerouslyByPassGVisor())
			require.Empty(t, logs.Records())
			requireGauge(t, 0)
		})
	}
}

func requireGauge(t *testing.T, expected float64) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "unkey_dangerous_flag_enabled" {
			continue
		}
		require.Len(t, family.GetMetric(), 1)
		metric := family.GetMetric()[0]
		require.Len(t, metric.GetLabel(), 1)
		require.Equal(t, "flag", metric.GetLabel()[0].GetName())
		require.Equal(t, "DANGEROUSLY_BYPASS_GVISOR_AND_GET_US_PWN3D", metric.GetLabel()[0].GetValue())
		require.Equal(t, expected, metric.GetGauge().GetValue())
		return
	}
	t.Fatal("dangerous flag gauge missing")
}
