package app

import (
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Program runs a Model against the terminal: raw-mode input, alternate screen,
// resize handling, and an async command runner.
type Program struct {
	model     Model
	altScreen bool
	in        *os.File
	out       *os.File
	// width/height track the real terminal size so paint never emits a frame
	// larger than the screen (which would wrap/scroll and garble the display).
	width  int
	height int
}

type Option func(*Program)

// WithAltScreen switches to the terminal's alternate screen buffer for the
// duration of the run, restoring the prior screen on exit.
func WithAltScreen() Option {
	return func(p *Program) { p.altScreen = true }
}

func NewProgram(m Model, opts ...Option) *Program {
	p := &Program{model: m, altScreen: false, in: os.Stdin, out: os.Stdout, width: 0, height: 0}
	for _, o := range opts {
		o(p)
	}
	return p
}

// Run drives the model until it quits or a terminating signal arrives. The
// terminal is always restored: on normal quit, on panic (via defer), and on
// SIGINT/SIGTERM (Ctrl+C is delivered in-band as a key, so the signal path is
// for external kills).
func (p *Program) Run() (Model, error) {
	fd := int(p.in.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return p.model, err
	}
	defer func() {
		_, _ = io.WriteString(p.out, enableWrap)
		if p.altScreen {
			_, _ = io.WriteString(p.out, exitAltScreen)
		}
		_, _ = io.WriteString(p.out, showCursor)
		_ = term.Restore(fd, oldState)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if p.altScreen {
		_, _ = io.WriteString(p.out, enterAltScreen)
	}
	_, _ = io.WriteString(p.out, hideCursor)
	_, _ = io.WriteString(p.out, disableWrap)

	// Size the model from the real terminal before the first paint so the
	// opening frame is laid out correctly, not at the default fallback size.
	outFd := int(p.out.Fd())
	if w, h, sizeErr := term.GetSize(outFd); sizeErr == nil {
		p.width, p.height = w, h
		p.model, _ = p.model.Update(WindowSizeMsg{Width: w, Height: h})
	}

	msgCh := make(chan Msg, 64)
	go readInput(p.in, msgCh)
	go watchResize(outFd, msgCh)

	p.exec(p.model.Init(), msgCh)

	lastFrame := ""
	lastFrame = p.paint(lastFrame)

	for {
		select {
		case <-sigCh:
			return p.model, nil
		case msg := <-msgCh:
			if msg == nil {
				continue
			}
			if _, ok := msg.(quitMsg); ok {
				return p.model, nil
			}
			if ws, ok := msg.(WindowSizeMsg); ok {
				p.width, p.height = ws.Width, ws.Height
			}
			var cmd Cmd
			p.model, cmd = p.model.Update(msg)
			p.exec(cmd, msgCh)
			lastFrame = p.paint(lastFrame)
		}
	}
}

// exec runs a command off the loop and delivers its result. Batches fan out to
// concurrent goroutines.
func (p *Program) exec(cmd Cmd, msgCh chan Msg) {
	if cmd == nil {
		return
	}
	go func() {
		p.deliver(cmd(), msgCh)
	}()
}

func (p *Program) deliver(msg Msg, msgCh chan Msg) {
	if msg == nil {
		return
	}
	if batch, ok := msg.(batchMsg); ok {
		for _, c := range batch {
			p.exec(c, msgCh)
		}
		return
	}
	msgCh <- msg
}

// paint repaints only when the frame changed, moving to home and clearing each
// line to the end (plus below) rather than clearing the whole screen, so
// steady-state redraws (spinner, ticks) do not flicker.
func (p *Program) paint(prev string) string {
	frame := p.model.View()
	if frame == prev {
		return prev
	}
	lines := strings.Split(frame, "\n")
	// Clamp to the physical screen: never emit a line wider than the terminal
	// or more rows than fit, so a mis-sized frame can't wrap or scroll and push
	// content off the left/top.
	if p.height > 0 && len(lines) > p.height {
		lines = lines[:p.height]
	}
	var b strings.Builder
	b.WriteString(cursorHome)
	for i, ln := range lines {
		if i > 0 {
			b.WriteString("\r\n")
		}
		// Clear the row to its end BEFORE writing content. Clearing after a
		// full-width line would erase its last cell: writing the final column
		// leaves the cursor in pending-wrap there, and \033[K erases from the
		// cursor onward, taking that character with it.
		b.WriteString(clearLineToEnd)
		if p.width > 0 {
			ln = clampWidth(ln, p.width)
		}
		b.WriteString(ln)
	}
	b.WriteString(clearScreenBelow)
	_, _ = io.WriteString(p.out, b.String())
	return frame
}

// clampWidth truncates a line to at most width visible cells, keeping ANSI
// escape sequences intact (they occupy no cells) and resetting styling if the
// line was cut mid-style.
func clampWidth(s string, width int) string {
	visible := 0
	inEsc := false
	truncated := false
	var b strings.Builder
	for _, r := range s {
		if inEsc {
			b.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if visible >= width {
			// Past the width budget: drop the rest (including any trailing
			// escapes) and close styling ourselves.
			truncated = true
			break
		}
		if r == '\x1b' {
			inEsc = true
			b.WriteRune(r)
			continue
		}
		b.WriteRune(r)
		visible++
	}
	if truncated {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}
