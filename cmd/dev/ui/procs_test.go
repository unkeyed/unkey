package ui

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// procsDir is relative to the working directory, so tests run in a temp dir.
func chdirTemp(t *testing.T) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(orig))
	})
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.Fail(t, "condition not met within "+timeout.String())
}

func TestManagedProcLifecycle(t *testing.T) {
	chdirTemp(t)

	err := startManagedProc("lifecycle", []string{"sh", "-c", "echo hello-from-proc; sleep 30"}, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		if st, ok := loadProcStatus("lifecycle"); ok && st.Running {
			_ = signalManagedProc(st, syscall.SIGKILL)
		}
	})

	rows := listProcs()
	require.Len(t, rows, 1)
	require.Equal(t, "lifecycle", rows[0].Name)
	require.True(t, rows[0].Running)

	// Starting the same name again must be rejected while it runs.
	err = startManagedProc("lifecycle", []string{"sh", "-c", "true"}, "")
	require.ErrorContains(t, err, "already running")

	waitFor(t, 5*time.Second, func() bool {
		lines, err := tailLog(procLogPath("lifecycle"), 10)
		return err == nil && strings.Contains(strings.Join(lines, "\n"), "hello-from-proc")
	})

	st, ok := loadProcStatus("lifecycle")
	require.True(t, ok)
	require.NoError(t, signalManagedProc(st, syscall.SIGTERM))
	waitFor(t, 5*time.Second, func() bool {
		st, ok := loadProcStatus("lifecycle")
		return ok && !st.Running
	})

	require.Equal(t, 1, clearExitedProcs())
	require.Empty(t, listProcs())
}

func TestManagedProcStdinFeed(t *testing.T) {
	chdirTemp(t)

	// Mirrors how nuke's read -p prompts are answered with blank lines.
	err := startManagedProc("prompted", []string{"sh", "-c", `read -r answer; echo "got:${answer:-empty}"`}, "\n\n")
	require.NoError(t, err)

	waitFor(t, 5*time.Second, func() bool {
		lines, err := tailLog(procLogPath("prompted"), 10)
		return err == nil && strings.Contains(strings.Join(lines, "\n"), "got:empty")
	})
}

func TestTailLogTruncatesToMaxLines(t *testing.T) {
	chdirTemp(t)
	require.NoError(t, os.MkdirAll(procsDir, 0o755))
	var b strings.Builder
	for range 100 {
		b.WriteString("line\n")
	}
	b.WriteString("last\n")
	require.NoError(t, os.WriteFile(procLogPath("big"), []byte(b.String()), 0o644))

	lines, err := tailLog(procLogPath("big"), 10)
	require.NoError(t, err)
	require.Len(t, lines, 10)
	require.Equal(t, "last", lines[len(lines)-1])
}
