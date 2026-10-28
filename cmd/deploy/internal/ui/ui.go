package ui

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/term"
)

type UI struct {
	mu          sync.Mutex
	writer      io.Writer
	terminal    bool
	color       bool
	spinnerStop chan struct{}
	spinnerDone chan struct{}
}

func NewUI() *UI {
	return newUI(os.Stderr, term.IsTerminal(int(os.Stderr.Fd())))
}

func newUI(writer io.Writer, terminal bool) *UI {
	terminal = terminal && os.Getenv("TERM") != "dumb"
	return &UI{
		mu:          sync.Mutex{},
		writer:      writer,
		terminal:    terminal,
		color:       terminal && os.Getenv("NO_COLOR") == "",
		spinnerStop: nil,
		spinnerDone: nil,
	}
}

func (ui *UI) StartSpinner(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.stopSpinner()
	message = Text(message)

	if !ui.terminal {
		fmt.Fprintln(ui.writer, message)
		return
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	ui.spinnerStop = stop
	ui.spinnerDone = done
	fmt.Fprintf(ui.writer, "\r⠋ %s", message)
	go func() {
		defer close(done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := 1
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fmt.Fprintf(ui.writer, "\r%s %s", frames[frame%len(frames)], message)
				frame++
			}
		}
	}()
}

func (ui *UI) StopSpinner(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.stopSpinner()
	message = Text(message)

	if !ui.terminal {
		fmt.Fprintln(ui.writer, message)
		return
	}

	if ui.color {
		fmt.Fprintf(ui.writer, "\033[32m✔\033[0m %s\n", message)
	} else {
		fmt.Fprintf(ui.writer, "✔ %s\n", message)
	}
}

func (ui *UI) Close() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.stopSpinner()
}

func (ui *UI) stopSpinner() {
	if ui.spinnerStop == nil {
		return
	}
	close(ui.spinnerStop)
	<-ui.spinnerDone
	ui.spinnerStop = nil
	ui.spinnerDone = nil
	fmt.Fprint(ui.writer, "\r\033[K")
}

// Text escapes control and format characters in externally supplied terminal text.
func Text(value string) string {
	var output strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			escaped := strconv.QuoteRune(r)
			output.WriteString(escaped[1 : len(escaped)-1])
		} else {
			output.WriteRune(r)
		}
	}
	return output.String()
}
