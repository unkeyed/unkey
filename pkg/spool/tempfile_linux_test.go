//go:build linux

package spool

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCreateUnlinked_LeavesNoDirectoryEntry guarantees that the fallback for
// filesystems without O_TMPFILE also stores bodies in private files without a
// name. Kernels with O_TMPFILE never reach this path in other tests.
func TestCreateUnlinked_LeavesNoDirectoryEntry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	file, err := createUnlinked(dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries)
	info, err := file.Stat()
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	_, err = file.Write([]byte("abc"))
	require.NoError(t, err)
}
