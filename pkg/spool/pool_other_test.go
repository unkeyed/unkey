//go:build !linux

package spool_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/spool"
)

// TestNewPool_FailsWithoutLinux guarantees that the pool does not claim
// disk-backed storage on systems where it cannot check the filesystem.
func TestNewPool_FailsWithoutLinux(t *testing.T) {
	pool, err := spool.NewPool(spool.Config{Dir: t.TempDir(), BodyBytesMax: 1, TotalBytesMax: 1, ConcurrentMax: 1})

	require.ErrorContains(t, err, "requires Linux")
	require.Nil(t, pool)
}
