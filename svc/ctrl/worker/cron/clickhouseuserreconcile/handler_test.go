package clickhouseuserreconcile_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/clickhouseuserreconcile"
)

func TestNewRequiresHeartbeat(t *testing.T) {
	_, err := clickhouseuserreconcile.New(clickhouseuserreconcile.Config{
		DB: &workspaceListDB{},
	})
	require.ErrorContains(t, err, "Heartbeat must not be nil",
		"a missing heartbeat must fail construction instead of silently skipping the ping")
}
