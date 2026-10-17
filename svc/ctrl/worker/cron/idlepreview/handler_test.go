package idlepreview

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type fakeDB struct {
	db.Database
}

func TestNewRequiresHeartbeat(t *testing.T) {
	_, err := New(Config{
		DB:         &fakeDB{},
		Clickhouse: clickhouse.NewNoop(),
	})
	require.ErrorContains(t, err, "Heartbeat must not be nil",
		"a missing heartbeat must fail construction instead of silently skipping the ping")
}
