package ui

import (
	"context"
	"os"
	"syscall"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

// Run starts the dev TUI.
func Run(ctx context.Context) error {
	loadDevEnvFiles()
	restoreLogs := logger.Quiet()
	restoreStderr := suppressStderrFD()
	defer func() {
		restoreStderr()
		restoreLogs()
	}()

	m := newAppModel()
	p := app.NewProgram(m, app.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	_ = ctx
	return nil
}

// suppressStderrFD redirects file descriptor 2 to /dev/null for the TUI session.
// pkg/logger and other libraries keep the stderr fd from init; reassigning
// os.Stderr alone does not stop them from corrupting the alt-screen buffer.
func suppressStderrFD() func() {
	stderrFd := int(os.Stderr.Fd())
	saved, err := syscall.Dup(stderrFd)
	if err != nil {
		return func() {}
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		_ = syscall.Close(saved)
		return func() {}
	}

	if err := syscall.Dup2(int(devNull.Fd()), stderrFd); err != nil {
		_ = devNull.Close()
		_ = syscall.Close(saved)
		return func() {}
	}

	return func() {
		_ = syscall.Dup2(saved, stderrFd)
		_ = syscall.Close(saved)
		_ = devNull.Close()
	}
}
