package tui

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Renderer writes styled output to a single destination. Create one per
// command invocation with [New]; all styling methods return their input
// unchanged when the destination does not support color.
type Renderer struct {
	w     io.Writer
	color bool
}

// New returns a Renderer that auto-detects color support on w.
func New(w io.Writer) *Renderer {
	return &Renderer{w: w, color: shouldColor(w)}
}

// NewWithColor returns a Renderer with color support forced on or off,
// bypassing terminal detection. Useful in tests.
func NewWithColor(w io.Writer, color bool) *Renderer {
	return &Renderer{w: w, color: color}
}

// shouldColor reports whether escape codes should be emitted: the writer must
// be a terminal, and the user must not have opted out via the NO_COLOR
// convention (https://no-color.org) or a dumb TERM.
func shouldColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// Println writes the arguments followed by a newline. Write errors are
// dropped: terminal output is best-effort and a failing stdout leaves the
// caller nothing useful to do.
func (r *Renderer) Println(args ...any) {
	_, _ = fmt.Fprintln(r.w, args...)
}

// Printf writes formatted output, dropping write errors like [Renderer.Println].
func (r *Renderer) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.w, format, args...)
}

// Blank writes an empty line, dropping write errors like [Renderer.Println].
func (r *Renderer) Blank() {
	_, _ = fmt.Fprintln(r.w)
}

// write emits a fully rendered block in a single Write call. Table and KV
// rendering assemble their output in memory first so a multi-row block does
// not turn into one syscall per row on unbuffered stdout.
func (r *Renderer) write(s string) {
	_, _ = io.WriteString(r.w, s)
}
