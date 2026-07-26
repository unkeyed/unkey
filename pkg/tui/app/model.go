// Package app is a small terminal-UI runtime in the Elm shape (Model/Update/
// View with async commands), built on the standard library plus
// golang.org/x/term. It covers a message loop, async commands, a periodic
// tick, keyboard input, the alternate screen, and terminal-resize handling.
package app

import "time"

// Msg is any event delivered to Model.Update.
type Msg any

// Cmd is a unit of async work. The runtime runs it on its own goroutine and
// feeds the returned Msg back into Update. A nil Cmd or a Cmd returning nil is
// a no-op.
type Cmd func() Msg

// Model is the application state. Init returns startup work, Update folds a
// message into new state plus follow-up work, and View renders the full frame.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}

// WindowSizeMsg reports the terminal size at startup and on every resize.
type WindowSizeMsg struct {
	Width  int
	Height int
}

type quitMsg struct{}

// Quit is a command that stops the program after the current update.
var Quit Cmd = func() Msg { return quitMsg{} }

// EnterAltScreen is a no-op command kept for source compatibility with callers
// that request the alternate screen from Init; the program enters it via
// WithAltScreen at startup.
var EnterAltScreen Cmd = func() Msg { return nil }

// batchMsg carries commands to run concurrently; the runtime expands it rather
// than delivering it to Update.
type batchMsg []Cmd

// Batch groups commands to run concurrently. nil commands are dropped.
func Batch(cmds ...Cmd) Cmd {
	filtered := make([]Cmd, 0, len(cmds))
	for _, c := range cmds {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return func() Msg { return batchMsg(filtered) }
}

// Tick delivers fn(now) after d has elapsed. It does not re-arm itself; return
// another Tick from Update to repeat.
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd {
	return func() Msg {
		time.Sleep(d)
		return fn(time.Now())
	}
}
