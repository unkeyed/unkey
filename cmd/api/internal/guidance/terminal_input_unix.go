//go:build linux || darwin

package guidance

import (
	"errors"
	"io"
	"time"

	"golang.org/x/sys/unix"
)

const terminalInputSupported = true

func (f *terminalForm) escapeReady() (bool, error) {
	if f.in.Buffered() > 0 {
		return true, nil
	}
	deadline := time.Now().Add(50 * time.Millisecond)
	fds := []unix.PollFd{{Fd: int32(f.fd), Events: unix.POLLIN, Revents: 0}}
	for {
		remaining := time.Until(deadline).Milliseconds()
		if remaining <= 0 {
			return false, nil
		}
		n, err := unix.Poll(fds, int(remaining))
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return n > 0, err
	}
}

func (f *terminalForm) loadingInput() error {
	for f.in.Buffered() > 0 {
		key, err := f.in.ReadByte()
		if err != nil {
			return err
		}
		if key == 3 || key == 4 {
			return errInterrupted
		}
	}
	fds := []unix.PollFd{{Fd: int32(f.fd), Events: unix.POLLIN, Revents: 0}}
	n, err := unix.Poll(fds, 20)
	if errors.Is(err, unix.EINTR) {
		return nil
	}
	if err != nil || n == 0 {
		return err
	}
	var input [1024]byte
	n, err = unix.Read(f.fd, input[:])
	if err != nil {
		return err
	}
	if n == 0 {
		return io.EOF
	}
	for _, key := range input[:n] {
		if key == 3 || key == 4 {
			return errInterrupted
		}
	}
	return nil
}
