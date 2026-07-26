package app

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// watchResize emits the terminal size once at startup and again on every
// SIGWINCH. fd is the output tty descriptor.
func watchResize(fd int, msgCh chan<- Msg) {
	sizeCh := make(chan os.Signal, 1)
	signal.Notify(sizeCh, syscall.SIGWINCH)

	emit := func() {
		if w, h, err := term.GetSize(fd); err == nil {
			msgCh <- WindowSizeMsg{Width: w, Height: h}
		}
	}
	emit()
	for range sizeCh {
		emit()
	}
}
