//go:build linux

package spool_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/spool"
	"golang.org/x/sys/unix"
)

// TestNewPool_RejectsInvalidConfig guarantees that a pool cannot start with
// limits that make its bounds meaningless.
func TestNewPool_RejectsInvalidConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tests := []struct {
		name string
		cfg  spool.Config
	}{
		{name: "relative Dir", cfg: spool.Config{Dir: "spool", BodyBytesMax: 1, TotalBytesMax: 1, ConcurrentMax: 1}},
		{name: "missing Dir", cfg: spool.Config{Dir: filepath.Join(dir, "missing"), BodyBytesMax: 1, TotalBytesMax: 1, ConcurrentMax: 1}},
		{name: "zero BodyBytesMax", cfg: spool.Config{Dir: dir, BodyBytesMax: 0, TotalBytesMax: 1, ConcurrentMax: 1}},
		{name: "TotalBytesMax below BodyBytesMax", cfg: spool.Config{Dir: dir, BodyBytesMax: 2, TotalBytesMax: 1, ConcurrentMax: 1}},
		{name: "zero ConcurrentMax", cfg: spool.Config{Dir: dir, BodyBytesMax: 1, TotalBytesMax: 1, ConcurrentMax: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pool, err := spool.NewPool(tt.cfg)
			require.Error(t, err)
			require.Nil(t, pool)
		})
	}
}

// TestNewPool_RejectsMemoryBackedDir guarantees that bodies are never stored
// on tmpfs, where they would count fully against process memory.
func TestNewPool_RejectsMemoryBackedDir(t *testing.T) {
	t.Parallel()
	var stat unix.Statfs_t
	if err := unix.Statfs("/dev/shm", &stat); err != nil || int64(stat.Type) != unix.TMPFS_MAGIC {
		t.Skip("/dev/shm is not a tmpfs mount on this machine")
	}

	pool, err := spool.NewPool(spool.Config{Dir: "/dev/shm", BodyBytesMax: 1, TotalBytesMax: 1, ConcurrentMax: 1})

	require.ErrorContains(t, err, "tmpfs")
	require.Nil(t, pool)
}
