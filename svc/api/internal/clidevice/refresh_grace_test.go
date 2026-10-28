package clidevice

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRefreshReuseGraceBounds(t *testing.T) {
	now := int64(1_000_000)
	rotated := sql.NullInt64{Int64: now, Valid: true}
	require.True(t, refreshReuseAllowed(rotated, now))

	rotated.Int64 = now - refreshReuseGrace.Milliseconds()
	require.True(t, refreshReuseAllowed(rotated, now))

	rotated.Int64 = now - refreshReuseGrace.Milliseconds() - 1
	require.False(t, refreshReuseAllowed(rotated, now))

	require.False(t, refreshReuseAllowed(sql.NullInt64{}, now))
	require.False(t, refreshReuseAllowed(sql.NullInt64{Int64: now + 1, Valid: true}, now))
}
