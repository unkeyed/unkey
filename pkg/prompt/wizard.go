package prompt

import (
	"fmt"
	"strings"
	"time"
)

// Wizard provides a multi-step prompt flow with visual progress tracking.
//
// Each prompt is prefixed with a progress indicator showing completed steps as
// green dots, the current step as a cyan dot, and remaining steps as gray circles.
// For example, a 4-step wizard on step 2 displays: [●●○○]
//
// Wizard wraps a [Prompt] instance and delegates to its methods, adding only the
// progress prefix. This means all the same input validation, defaults, and error
// handling from the underlying prompt methods apply.
//
// Use [Prompt.Wizard] to create a new wizard, then call the same methods you would
// on [Prompt] (String, Int, Select, etc.). The wizard automatically advances after
// each successful prompt.
type Wizard struct {
	prompt  *Prompt
	total   int
	current int
}

// Wizard creates a new wizard that tracks progress through totalSteps steps.
//
// Each prompt method on the returned [Wizard] displays a progress indicator before
// the label. The indicator uses colored dots: green for completed steps, cyan for
// the current step, and gray for remaining steps.
//
// The wizard automatically advances after each successful prompt. Use [Wizard.Skip]
// to advance without prompting when a step is conditionally not needed.
func (p *Prompt) Wizard(totalSteps int) *Wizard {
	return &Wizard{
		prompt:  p,
		total:   totalSteps,
		current: 0,
	}
}

// Skip advances the wizard to the next step without displaying a prompt.
//
// Use this when a step is conditionally not needed based on previous answers.
// For example, skip a "Python version" step if the user selected Go as their language.
func (w *Wizard) Skip() {
	w.current++
}

// prefix builds the progress indicator string for the current step.
// Returns a string like "[●●○○] " with appropriate colors.
func (w *Wizard) prefix() string {
	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < w.total; i++ {
		if i < w.current {
			b.WriteString(colorGreen + "●" + colorReset)
		} else if i == w.current {
			b.WriteString(colorCyan + "●" + colorReset)
		} else {
			b.WriteString(colorDim + "○" + colorReset)
		}
	}
	b.WriteString("] ")
	return b.String()
}

// advance moves the wizard to the next step.
func (w *Wizard) advance() {
	w.current++
}

// String prompts for text input at the current step and advances on success.
// See [Prompt.String] for parameter details.
func (w *Wizard) String(label string, defaultValue ...string) (string, error) {
	result, err := w.prompt.String(w.prefix()+label, defaultValue...)
	if err == nil {
		w.advance()
	}
	return result, err
}

// Int prompts for integer input at the current step and advances on success.
// Accepts human-readable suffixes (k, m, b, t) like [Prompt.Int].
func (w *Wizard) Int(label string, defaultValue ...int) (int, error) {
	result, err := w.prompt.Int(w.prefix()+label, defaultValue...)
	if err == nil {
		w.advance()
	}
	return result, err
}

// Float prompts for floating-point input at the current step and advances on success.
// Accepts human-readable suffixes (k, m, b, t) like [Prompt.Float].
func (w *Wizard) Float(label string, defaultValue ...float64) (float64, error) {
	result, err := w.prompt.Float(w.prefix()+label, defaultValue...)
	if err == nil {
		w.advance()
	}
	return result, err
}

// Select displays a single-choice menu at the current step and advances on success.
// See [Prompt.Select] for parameter details.
func (w *Wizard) Select(label string, options map[string]string, defaultKey ...string) (string, error) {
	result, err := w.prompt.Select(w.prefix()+label, options, defaultKey...)
	if err == nil {
		w.advance()
	}
	return result, err
}

// MultiSelect displays a multi-choice menu at the current step and advances on success.
// See [Prompt.MultiSelect] for parameter details.
func (w *Wizard) MultiSelect(label string, options map[string]string, defaultKeys ...string) ([]string, error) {
	result, err := w.prompt.MultiSelect(w.prefix()+label, options, defaultKeys...)
	if err == nil {
		w.advance()
	}
	return result, err
}

// Date displays an interactive date picker at the current step and advances on success.
// See [Prompt.Date] for parameter details.
func (w *Wizard) Date(label string, defaultValue ...time.Time) (time.Time, error) {
	result, err := w.prompt.Date(w.prefix()+label, defaultValue...)
	if err == nil {
		w.advance()
	}
	return result, err
}

// Time displays an interactive time picker at the current step and advances on success.
// See [Prompt.Time] for parameter details.
func (w *Wizard) Time(label string, minuteStep int, defaultValue ...time.Time) (time.Time, error) {
	result, err := w.prompt.Time(w.prefix()+label, minuteStep, defaultValue...)
	if err == nil {
		w.advance()
	}
	return result, err
}

// DateTime displays a combined date/time picker at the current step and advances on success.
// See [Prompt.DateTime] for parameter details.
func (w *Wizard) DateTime(label string, minuteStep int, defaultValue ...time.Time) (time.Time, error) {
	result, err := w.prompt.DateTime(w.prefix()+label, minuteStep, defaultValue...)
	if err == nil {
		w.advance()
	}
	return result, err
}

// Done prints a completion message in green.
//
// This is optional but provides a clean visual ending to the wizard flow.
// The message is printed on a new line with green text.
func (w *Wizard) Done(message string) {
	_, _ = fmt.Fprintf(w.prompt.out, "\n%s%s%s\n", colorGreen, message, colorReset)
}

// Current returns the current step number (0-indexed).
func (w *Wizard) Current() int {
	return w.current
}

// Total returns the total number of steps.
func (w *Wizard) Total() int {
	return w.total
}
