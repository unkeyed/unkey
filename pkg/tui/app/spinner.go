package app

import (
	"sync/atomic"
	"time"
)

const spinnerFPS = time.Second / 10

var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

// spinnerIDs hands out a unique id per spinner so that, when several spinners
// animate at once, each only advances on its own ticks. Without this a shared
// tick would be consumed by whichever spinner's owner is reached first.
var spinnerIDs atomic.Int64

// SpinnerTickMsg advances the spinner whose ID it carries.
type SpinnerTickMsg struct{ ID int64 }

// Spinner is a minimal frame-cycling spinner. It is a value type; Update
// returns the advanced copy.
type Spinner struct {
	id     int64
	frames []string
	index  int
}

func NewSpinner() Spinner {
	return Spinner{id: spinnerIDs.Add(1), frames: spinnerFrames, index: 0}
}

// ID identifies which ticks belong to this spinner.
func (s Spinner) ID() int64 { return s.id }

func (s Spinner) View() string {
	if len(s.frames) == 0 {
		return ""
	}
	return s.frames[s.index%len(s.frames)]
}

// Tick schedules the next frame advance for this spinner.
func (s Spinner) Tick() Cmd {
	id := s.id
	return Tick(spinnerFPS, func(time.Time) Msg { return SpinnerTickMsg{ID: id} })
}

// Update advances the frame and re-arms only for this spinner's own ticks;
// other messages (including another spinner's ticks) are ignored.
func (s Spinner) Update(msg Msg) (Spinner, Cmd) {
	if tick, ok := msg.(SpinnerTickMsg); ok && tick.ID == s.id {
		s.index++
		return s, s.Tick()
	}
	return s, nil
}

// WithSpinnerTick batches a spinner's next tick with an async command so the
// spinner animates while that command runs.
func WithSpinnerTick(sp Spinner, cmd Cmd) Cmd {
	if cmd == nil {
		return sp.Tick()
	}
	return Batch(sp.Tick(), cmd)
}

// HandleSpinnerTick advances sp only for its own tick, returning whether it
// consumed the message. This lets a busy pane ignore another pane's ticks.
func HandleSpinnerTick(sp Spinner, msg Msg) (Spinner, Cmd, bool) {
	tick, ok := msg.(SpinnerTickMsg)
	if !ok || tick.ID != sp.ID() {
		return sp, nil, false
	}
	next, cmd := sp.Update(msg)
	return next, cmd, true
}
