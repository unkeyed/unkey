package guidance

import "golang.org/x/sys/unix"

func flushTerminalInput(fd int) error {
	const readQueue = 1
	return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, readQueue)
}
