---
title: prompt
description: "provides interactive terminal prompts for CLI applications"
---

Package prompt provides interactive terminal prompts for CLI applications.

The package implements common prompt patterns including text input, numeric input, single and multi-selection menus, and date/time pickers. It uses only the Go standard library and golang.org/x/term for raw terminal mode, avoiding external dependencies like survey or promptui to keep the dependency footprint minimal.

ANSI escape codes handle cursor movement and styling, which work on most modern terminals including macOS Terminal, iTerm2, Windows Terminal, and Linux terminals. The package may not render correctly on older terminals without ANSI support.

### Key Types

The main entry point is \[Prompt], created with \[New]. It provides methods for different input types: \[Prompt.String], \[Prompt.Int], \[Prompt.Float] for text and numeric input, \[Prompt.Select] and \[Prompt.MultiSelect] for selection menus, and \[Prompt.Date], \[Prompt.Time], \[Prompt.DateTime] for date/time pickers.

For multi-step workflows, \[Wizard] wraps a \[Prompt] and adds progress tracking with a visual indicator showing completed, current, and remaining steps.

### Basic Usage

Create a \[Prompt] and call its methods. Without options, it uses os.Stdin and os.Stdout:

	p := prompt.New()
	name, err := p.String("Enter your name")
	if err != nil {
	    return err
	}

All prompt methods accept optional default values that are returned when the user enters nothing:

	name, _ := p.String("Name", "Anonymous")
	age, _ := p.Int("Age", 25)
	port, _ := p.Int("Port", 8080)

### Selection Menus

Single selection with arrow keys returns the selected key:

	key, err := p.Select("Pick environment", map[string]string{
	    "dev":  "Development",
	    "stg":  "Staging",
	    "prod": "Production",
	}, "dev")  // "dev" is pre-selected

Multiple selection with space to toggle returns all selected keys:

	keys, err := p.MultiSelect("Enable features", map[string]string{
	    "log":     "Logging",
	    "metrics": "Metrics",
	    "trace":   "Tracing",
	}, "log", "metrics")  // pre-selected defaults

### Human-Readable Numbers

\[Prompt.Int] and \[Prompt.Float] accept human-readable suffixes for large numbers:

	// User can type "1.5k" instead of "1500"
	limit, _ := p.Int("Rate limit per minute")  // accepts: 1k, 1.5m, 2b, 1t

### Date and Time Pickers

Interactive pickers for dates and times with keyboard navigation:

	date, _ := p.Date("Start date")           // calendar with arrow key navigation
	time, _ := p.Time("Meeting time", 15)     // 15-minute increments
	dt, _ := p.DateTime("Deadline", 30)       // combined picker

For parsing date/time strings without interactive prompts, use \[ParseDate], \[ParseTime], or \[ParseDateTime].

### Multi-Step Wizards

\[Wizard] tracks progress through multiple prompts with a visual indicator:

	wiz := p.Wizard(3)  // 3 total steps
	name, _ := wiz.String("Project name")     // shows [●○○]
	env, _ := wiz.Select("Environment", opts) // shows [●●○]
	confirm, _ := wiz.Select("Confirm", opts) // shows [●●●]
	wiz.Done("Project created!")

### Testing

For testing, inject custom readers and writers using functional options:

	p := prompt.New(
	    prompt.WithReader(strings.NewReader("test input\n")),
	    prompt.WithWriter(&bytes.Buffer{}),
	)

Note that \[Prompt.Select], \[Prompt.MultiSelect], and the date/time pickers require a real terminal for raw mode. In tests, these methods return an error when the input is not a TTY. Test the error path or use integration tests with a real terminal.

### Terminal Handling

Methods that require user navigation (\[Prompt.Select], \[Prompt.MultiSelect], \[Prompt.Date], \[Prompt.Time], \[Prompt.DateTime]) temporarily switch the terminal to raw mode to capture individual keypresses. The original terminal state is always restored on return, including on error or interrupt (Ctrl+C).

If the terminal state cannot be modified (e.g., when stdin is not a TTY or in a CI environment), these methods return an error immediately.

## Constants

ANSI escape sequences for terminal control.

These codes follow the ANSI X3.64 / ECMA-48 standard and work on most modern terminals (macOS Terminal, iTerm2, Windows Terminal, Linux terminals).

Escape sequences start with ESC (0x1B or \\033 in octal), followed by '\[' (CSI - Control Sequence Introducer), then parameters and a command letter.

Format: ESC \[ \<params> \<command>

References:

  - [https://en.wikipedia.org/wiki/ANSI\_escape\_code](https://en.wikipedia.org/wiki/ANSI_escape_code)
  - [https://invisible-island.net/xterm/ctlseqs/ctlseqs.html](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
```go
const (
	// Cursor visibility controls.
	// These use DEC Private Mode sequences (ESC [ ? <mode> h/l).
	// Mode 25 controls cursor visibility: 'l' = hide, 'h' = show.
	hideCursor = "\033[?25l" // ESC [ ? 25 l - hide cursor
	showCursor = "\033[?25h" // ESC [ ? 25 h - show cursor

	// Line editing.
	// clearLine erases from cursor to end of line (ESC [ K or ESC [ 0 K).
	// We combine with carriageReturn to clear the entire line.
	clearLine      = "\033[K" // ESC [ K - erase from cursor to end of line
	carriageReturn = "\r"     // Move cursor to beginning of current line (ASCII 13)
	newLine        = "\r\n"   // Carriage return + line feed for new line in raw mode

	// SGR (Select Graphic Rendition) sequences for text colors.
	// Format: ESC [ <color-code> m
	// 30-37 are foreground colors, 0 resets all attributes.
	colorCyan  = "\033[36m" // ESC [ 36 m - cyan foreground (color code 36)
	colorGreen = "\033[32m" // ESC [ 32 m - green foreground (color code 32)
	colorRed   = "\033[31m" // ESC [ 31 m - red foreground (color code 31)
	colorDim   = "\033[90m" // ESC [ 90 m - bright black (gray) foreground
	colorReset = "\033[0m"  // ESC [ 0 m - reset all text attributes to default

	// Unicode symbols for visual indicators.
	pointerSymbol    = "❯" // Shows which option the cursor is on
	selectedSymbol   = "●" // Filled circle for selected items in MultiSelect
	unselectedSymbol = "○" // Empty circle for unselected items in MultiSelect
)
```

Extended key codes for date/time pickers. These are the final bytes of escape sequences for special keys. PgUp/PgDn/Home/End send 4-byte sequences: ESC \[ \<code> ~
```go
const (
	keyTab   = '\t' // Tab character (9)
	keyPgUp  = '5'  // ESC [ 5 ~ - Page Up
	keyPgDn  = '6'  // ESC [ 6 ~ - Page Down
	keyHome  = 'H'  // ESC [ H or ESC O H - Home
	keyEnd   = 'F'  // ESC [ F or ESC O F - End
	keyLeft  = 'D'  // ESC [ D - Left arrow
	keyRight = 'C'  // ESC [ C - Right arrow
)
```

Platform-specific int bounds. On 64-bit systems, int is 64 bits. On 32-bit systems, int is 32 bits. We use these constants to check for overflow after multiplication.
```go
const (
	maxInt = int(^uint(0) >> 1) // Max int value for current platform
	minInt = -maxInt - 1        // Min int value for current platform
)
```

Key codes for detecting user input in raw terminal mode. When the terminal is in raw mode, arrow keys send a 3-byte escape sequence:

  - Byte 0: ESC (27 or 0x1B)
  - Byte 1: '\[' (91 or 0x5B)
  - Byte 2: Direction code ('A'=up, 'B'=down, 'C'=right, 'D'=left)

Regular keys send their ASCII value as a single byte.
```go
const (
	keyUp    = 'A'  // Third byte of arrow up escape sequence: ESC [ A
	keyDown  = 'B'  // Third byte of arrow down escape sequence: ESC [ B
	keyEnter = '\r' // Carriage return (13), sent when Enter is pressed in raw mode
	keySpace = ' '  // Space character (32)
)
```


## Variables

Suffix multipliers for human-readable number notation. These allow users to enter values like "1k" instead of "1000".

Supported suffixes (case-insensitive):

  - k, K: thousand (10^3 = 1,000)
  - m, M: million (10^6 = 1,000,000)
  - b, B: billion (10^9 = 1,000,000,000)
  - t, T: trillion (10^12 = 1,000,000,000,000)
```go
var suffixMultipliers = map[byte]int64{
	'k': 1_000,
	'K': 1_000,
	'm': 1_000_000,
	'M': 1_000_000,
	'b': 1_000_000_000,
	'B': 1_000_000_000,
	't': 1_000_000_000_000,
	'T': 1_000_000_000_000,
}
```


## Functions

### func ParseDate

```go
func ParseDate(s string) (time.Time, error)
```

ParseDate parses a human-readable date string into a time.Time value. The returned time is set to noon UTC to avoid DST edge cases.

Supported formats:

  - today, tomorrow, yesterday (case-insensitive)
  - \+1d, -2w, +3m, -1y (relative offsets: d=day, w=week, m=month, y=year)
  - 2024-01-15 (ISO format YYYY-MM-DD)
  - 01/15/2024 or 1/15/2024 (US format MM/DD/YYYY)
  - 15\.01.2024 (European format DD.MM.YYYY)

Returns an error if the input cannot be parsed.

### func ParseDateTime

```go
func ParseDateTime(s string) (time.Time, error)
```

ParseDateTime parses a combined date and time string. The date and time parts should be separated by a space or 'T' (ISO 8601).

Supported formats:

  - 2024-01-15 14:30
  - 2024-01-15T14:30
  - today 2pm
  - tomorrow 14:30
  - \+1d 9:00am

Returns the parsed time, or an error if parsing fails.

### func ParseTime

```go
func ParseTime(s string) (int, int, error)
```

ParseTime parses a human-readable time string into hour and minute values.

Supported formats:

  - now (current time)
  - 14:30, 14:30:00 (24-hour format HH:MM or HH:MM:SS)
  - 2:30pm, 2:30 PM, 2:30PM (12-hour format with am/pm)
  - 1430 (military time HHMM)

Returns hour (0-23) and minute (0-59), or an error if parsing fails.


## Types

### type Option

```go
type Option func(*Prompt)
```

Option configures a Prompt instance.

#### func WithReader

```go
func WithReader(r io.Reader) Option
```

WithReader sets the input reader. Defaults to os.Stdin.

#### func WithWriter

```go
func WithWriter(w io.Writer) Option
```

WithWriter sets the output writer. Defaults to os.Stdout.

### type Prompt

```go
type Prompt struct {
	in  io.Reader
	out io.Writer
	fd  int // file descriptor for raw mode, -1 if not a terminal
}
```

Prompt manages interactive terminal prompts with configurable I/O.

#### func New

```go
func New(opts ...Option) *Prompt
```

New creates a Prompt with the given options. Without options, uses os.Stdin and os.Stdout.

#### func (Prompt) Date

```go
func (p *Prompt) Date(label string, defaultValue ...time.Time) (time.Time, error)
```

Date displays an interactive calendar picker and returns the selected date. The user navigates with arrow keys (day/week), PgUp/PgDn (month), and confirms with Enter. If a default date is provided, the calendar opens to that date; otherwise uses today.

Smart parsing shortcuts are accepted if typed instead of navigating:

  - today, tomorrow, yesterday
  - \+1d, -1w, +2m (relative offsets: d=day, w=week, m=month, y=year)
  - 2024-01-15 (ISO format YYYY-MM-DD)

Returns the selected date at noon UTC to avoid DST edge cases.

#### func (Prompt) DateTime

```go
func (p *Prompt) DateTime(label string, minuteStep int, defaultValue ...time.Time) (time.Time, error)
```

DateTime displays a combined date and time picker. The user switches between the calendar and time picker with Tab, navigates within each using arrow keys, and confirms with Enter. If a default is provided, both pickers start at that value; otherwise uses now.

Accepts all smart parsing shortcuts from both Date and Time pickers. The minuteStep parameter controls the minute increment (default 5 if 0).

#### func (Prompt) Float

```go
func (p *Prompt) Float(label string, defaultValue ...float64) (float64, error)
```

Float prompts the user for a floating-point value. If a default is provided and the user enters nothing, the default is returned.

Accepts human-readable suffixes for large numbers:

  - k, K: thousand (1.5k = 1,500.0)
  - m, M: million (2.5m = 2,500,000.0)
  - b, B: billion (1b = 1,000,000,000.0)
  - t, T: trillion (1t = 1,000,000,000,000.0)

Returns an error if the input cannot be parsed.

#### func (Prompt) Int

```go
func (p *Prompt) Int(label string, defaultValue ...int) (int, error)
```

Int prompts the user for an integer value. If a default is provided and the user enters nothing, the default is returned.

Accepts human-readable suffixes for large numbers:

  - k, K: thousand (1k = 1,000)
  - m, M: million (1m = 1,000,000)
  - b, B: billion (1b = 1,000,000,000)
  - t, T: trillion (1t = 1,000,000,000,000)

Decimal values are supported with suffixes if the result is a whole number: "1.5k" = 1500 (valid), "1.5" without suffix = error.

Returns an error if the input cannot be parsed, is not a whole number, or would overflow.

#### func (Prompt) MultiSelect

```go
func (p *Prompt) MultiSelect(label string, options map[string]string, defaultKeys ...string) ([]string, error)
```

MultiSelect displays a multi-choice menu and returns the selected keys. The options map keys are internal codes returned on selection; values are display labels. The user navigates with arrow keys, toggles selection with Space, and confirms with Enter. If default keys are provided, those options are pre-selected.

Returns an error if the terminal cannot be switched to raw mode (e.g., stdin is not a TTY) or if the user interrupts with Ctrl+C.

#### func (Prompt) Select

```go
func (p *Prompt) Select(label string, options map[string]string, defaultKey ...string) (string, error)
```

Select displays a single-choice menu and returns the selected key. The options map keys are internal codes returned on selection; values are display labels. The user navigates with arrow keys and confirms with Enter. If a default key is provided, that option is pre-selected.

Returns an error if the terminal cannot be switched to raw mode (e.g., stdin is not a TTY) or if the user interrupts with Ctrl+C.

#### func (Prompt) String

```go
func (p *Prompt) String(label string, defaultValue ...string) (string, error)
```

String prompts the user for text input and returns the trimmed response. The label is displayed followed by a colon and space. Input is read until the user presses Enter. If a default is provided and the user enters nothing, the default is returned.

#### func (Prompt) Time

```go
func (p *Prompt) Time(label string, minuteStep int, defaultValue ...time.Time) (time.Time, error)
```

Time displays an interactive time picker with spinner-style selection. The user navigates between hour and minute with Left/Right arrows, adjusts values with Up/Down arrows, and confirms with Enter. If a default time is provided, the picker starts at that time; otherwise uses current time.

Smart parsing shortcuts are accepted:

  - 14:30, 2:30pm, 2:30 PM (various time formats)
  - now (current time)

The minuteStep parameter controls the increment/decrement step for minutes (default 5 if 0).

#### func (Prompt) Wizard

```go
func (p *Prompt) Wizard(totalSteps int) *Wizard
```

Wizard creates a new wizard that tracks progress through totalSteps steps.

Each prompt method on the returned \[Wizard] displays a progress indicator before the label. The indicator uses colored dots: green for completed steps, cyan for the current step, and gray for remaining steps.

The wizard automatically advances after each successful prompt. Use \[Wizard.Skip] to advance without prompting when a step is conditionally not needed.

### type Wizard

```go
type Wizard struct {
	prompt  *Prompt
	total   int
	current int
}
```

Wizard provides a multi-step prompt flow with visual progress tracking.

Each prompt is prefixed with a progress indicator showing completed steps as green dots, the current step as a cyan dot, and remaining steps as gray circles. For example, a 4-step wizard on step 2 displays: \[●●○○]

Wizard wraps a \[Prompt] instance and delegates to its methods, adding only the progress prefix. This means all the same input validation, defaults, and error handling from the underlying prompt methods apply.

Use \[Prompt.Wizard] to create a new wizard, then call the same methods you would on \[Prompt] (String, Int, Select, etc.). The wizard automatically advances after each successful prompt.

#### func (Wizard) Current

```go
func (w *Wizard) Current() int
```

Current returns the current step number (0-indexed).

#### func (Wizard) Date

```go
func (w *Wizard) Date(label string, defaultValue ...time.Time) (time.Time, error)
```

Date displays an interactive date picker at the current step and advances on success. See \[Prompt.Date] for parameter details.

#### func (Wizard) DateTime

```go
func (w *Wizard) DateTime(label string, minuteStep int, defaultValue ...time.Time) (time.Time, error)
```

DateTime displays a combined date/time picker at the current step and advances on success. See \[Prompt.DateTime] for parameter details.

#### func (Wizard) Done

```go
func (w *Wizard) Done(message string)
```

Done prints a completion message in green.

This is optional but provides a clean visual ending to the wizard flow. The message is printed on a new line with green text.

#### func (Wizard) Float

```go
func (w *Wizard) Float(label string, defaultValue ...float64) (float64, error)
```

Float prompts for floating-point input at the current step and advances on success. Accepts human-readable suffixes (k, m, b, t) like \[Prompt.Float].

#### func (Wizard) Int

```go
func (w *Wizard) Int(label string, defaultValue ...int) (int, error)
```

Int prompts for integer input at the current step and advances on success. Accepts human-readable suffixes (k, m, b, t) like \[Prompt.Int].

#### func (Wizard) MultiSelect

```go
func (w *Wizard) MultiSelect(label string, options map[string]string, defaultKeys ...string) ([]string, error)
```

MultiSelect displays a multi-choice menu at the current step and advances on success. See \[Prompt.MultiSelect] for parameter details.

#### func (Wizard) Select

```go
func (w *Wizard) Select(label string, options map[string]string, defaultKey ...string) (string, error)
```

Select displays a single-choice menu at the current step and advances on success. See \[Prompt.Select] for parameter details.

#### func (Wizard) Skip

```go
func (w *Wizard) Skip()
```

Skip advances the wizard to the next step without displaying a prompt.

Use this when a step is conditionally not needed based on previous answers. For example, skip a "Python version" step if the user selected Go as their language.

#### func (Wizard) String

```go
func (w *Wizard) String(label string, defaultValue ...string) (string, error)
```

String prompts for text input at the current step and advances on success. See \[Prompt.String] for parameter details.

#### func (Wizard) Time

```go
func (w *Wizard) Time(label string, minuteStep int, defaultValue ...time.Time) (time.Time, error)
```

Time displays an interactive time picker at the current step and advances on success. See \[Prompt.Time] for parameter details.

#### func (Wizard) Total

```go
func (w *Wizard) Total() int
```

Total returns the total number of steps.

