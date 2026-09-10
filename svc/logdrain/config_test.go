package logdrain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/logdrain"
)

const dependencies = `
region = "test"
database = "test"
[clickhouse]
url = "http://clickhouse"
[vault]
url = "http://vault"
token = "test"
`

func TestConfig_AuditTimingDefaultsAndOverrides(t *testing.T) {
	cfg, err := config.Load[logdrain.Config](dependencies)
	require.NoError(t, err)
	require.Equal(t, time.Minute, cfg.PollInterval)
	require.Equal(t, 5*time.Minute, cfg.WatermarkLag)

	cfg, err = config.Load[logdrain.Config](`
poll_interval = "90s"
watermark_lag = "7m"
` + dependencies)
	require.NoError(t, err)
	require.Equal(t, 90*time.Second, cfg.PollInterval)
	require.Equal(t, 7*time.Minute, cfg.WatermarkLag)
}

func TestConfig_RejectsInvalidAuditTiming(t *testing.T) {
	for _, setting := range []string{
		`poll_interval = "-1s"`,
		`poll_interval = "1ns"`,
		`watermark_lag = "-1s"`,
	} {
		t.Run(setting, func(t *testing.T) {
			_, err := config.Load[logdrain.Config](setting + "\n" + dependencies)
			require.Error(t, err)
		})
	}
}
