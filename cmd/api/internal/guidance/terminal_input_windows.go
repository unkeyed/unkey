package guidance

import (
	"errors"

	"golang.org/x/sys/windows"
)

// Console readiness includes non-key events, so it cannot bound a ReadConsole call.
const terminalInputSupported = false

func (f *terminalForm) escapeReady() (bool, error) {
	return false, errors.ErrUnsupported
}

func (f *terminalForm) loadingInput() error {
	return errors.ErrUnsupported
}

func flushTerminalInput(fd int) error {
	return windows.FlushConsoleInputBuffer(windows.Handle(fd))
}
