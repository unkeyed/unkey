// Package prompt provides interactive terminal prompts for CLI applications.
//
// The package implements common prompt patterns including text input, numeric input,
// single and multi-selection menus, and date/time pickers. It uses only the Go standard
// library and golang.org/x/term for raw terminal mode, avoiding external dependencies
// like survey or promptui to keep the dependency footprint minimal.
//
// ANSI escape codes handle cursor movement and styling, which work on most modern
// terminals including macOS Terminal, iTerm2, Windows Terminal, and Linux terminals.
// The package may not render correctly on older terminals without ANSI support.
//
// # Key Types
//
// The main entry point is [Prompt], created with [New]. It provides methods for
// different input types: [Prompt.String], [Prompt.Int], [Prompt.Float] for text and
// numeric input, [Prompt.Select] and [Prompt.MultiSelect] for selection menus, and
// [Prompt.Date], [Prompt.Time], [Prompt.DateTime] for date/time pickers.
//
// For multi-step workflows, [Wizard] wraps a [Prompt] and adds progress tracking
// with a visual indicator showing completed, current, and remaining steps.
//
// # Basic Usage
//
// Create a [Prompt] and call its methods. Without options, it uses os.Stdin and os.Stdout:
//
//	p := prompt.New()
//	name, err := p.String("Enter your name")
//	if err != nil {
//	    return err
//	}
//
// All prompt methods accept optional default values that are returned when the user
// enters nothing:
//
//	name, _ := p.String("Name", "Anonymous")
//	age, _ := p.Int("Age", 25)
//	port, _ := p.Int("Port", 8080)
//
// # Selection Menus
//
// Single selection with arrow keys returns the selected key:
//
//	key, err := p.Select("Pick environment", map[string]string{
//	    "dev":  "Development",
//	    "stg":  "Staging",
//	    "prod": "Production",
//	}, "dev")  // "dev" is pre-selected
//
// Multiple selection with space to toggle returns all selected keys:
//
//	keys, err := p.MultiSelect("Enable features", map[string]string{
//	    "log":     "Logging",
//	    "metrics": "Metrics",
//	    "trace":   "Tracing",
//	}, "log", "metrics")  // pre-selected defaults
//
// # Human-Readable Numbers
//
// [Prompt.Int] and [Prompt.Float] accept human-readable suffixes for large numbers:
//
//	// User can type "1.5k" instead of "1500"
//	limit, _ := p.Int("Rate limit per minute")  // accepts: 1k, 1.5m, 2b, 1t
//
// # Date and Time Pickers
//
// Interactive pickers for dates and times with keyboard navigation:
//
//	date, _ := p.Date("Start date")           // calendar with arrow key navigation
//	time, _ := p.Time("Meeting time", 15)     // 15-minute increments
//	dt, _ := p.DateTime("Deadline", 30)       // combined picker
//
// For parsing date/time strings without interactive prompts, use [ParseDate],
// [ParseTime], or [ParseDateTime].
//
// # Multi-Step Wizards
//
// [Wizard] tracks progress through multiple prompts with a visual indicator:
//
//	wiz := p.Wizard(3)  // 3 total steps
//	name, _ := wiz.String("Project name")     // shows [●○○]
//	env, _ := wiz.Select("Environment", opts) // shows [●●○]
//	confirm, _ := wiz.Select("Confirm", opts) // shows [●●●]
//	wiz.Done("Project created!")
//
// # Testing
//
// For testing, inject custom readers and writers using functional options:
//
//	p := prompt.New(
//	    prompt.WithReader(strings.NewReader("test input\n")),
//	    prompt.WithWriter(&bytes.Buffer{}),
//	)
//
// Note that [Prompt.Select], [Prompt.MultiSelect], and the date/time pickers require
// a real terminal for raw mode. In tests, these methods return an error when the
// input is not a TTY. Test the error path or use integration tests with a real terminal.
//
// # Terminal Handling
//
// Methods that require user navigation ([Prompt.Select], [Prompt.MultiSelect],
// [Prompt.Date], [Prompt.Time], [Prompt.DateTime]) temporarily switch the terminal
// to raw mode to capture individual keypresses. The original terminal state is
// always restored on return, including on error or interrupt (Ctrl+C).
//
// If the terminal state cannot be modified (e.g., when stdin is not a TTY or in
// a CI environment), these methods return an error immediately.
package prompt
