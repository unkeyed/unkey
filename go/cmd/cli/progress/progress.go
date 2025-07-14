// Package progress provides reusable animated progress tracking for CLI operations
package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Colors for different states
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// Animation characters
var (
	SpinnerChars  = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	DotsChars     = []string{"", ".", "..", "..."}
	ProgressChars = []string{"▱", "▰"}
	PulseChars    = []string{"◐", "◓", "◑", "◒"}
)

// Status represents the state of a tracked item
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusSkipped   Status = "skipped"
)

// Step represents a single step in a process
type Step struct {
	ID        string
	Name      string
	Status    Status
	Message   string
	Error     string
	StartTime time.Time
	EndTime   time.Time
	Active    bool
	Progress  float64 // 0.0 to 1.0 for progress bars
	metadata  map[string]interface{}
	mu        sync.RWMutex
}

// SetMetadata sets custom metadata for the step
func (s *Step) SetMetadata(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.metadata == nil {
		s.metadata = make(map[string]interface{})
	}
	s.metadata[key] = value
}

// GetMetadata gets custom metadata for the step
func (s *Step) GetMetadata(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.metadata == nil {
		return nil, false
	}
	val, exists := s.metadata[key]
	return val, exists
}

// Duration returns the duration of the step
func (s *Step) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.EndTime.IsZero() {
		if s.StartTime.IsZero() {
			return 0
		}
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// Tracker manages animated progress tracking
type Tracker struct {
	title     string
	steps     map[string]*Step
	stepOrder []string
	animation animationState
	done      chan struct{}
	running   bool
	mu        sync.RWMutex
	options   TrackerOptions
}

type animationState struct {
	frame      int
	lastUpdate time.Time
}

// TrackerOptions configures the tracker behavior
type TrackerOptions struct {
	ShowElapsed    bool          // Show elapsed time for running steps
	ShowDuration   bool          // Show duration for completed steps
	ShowProgress   bool          // Show progress bars when available
	AnimationSpeed time.Duration // Animation update interval
	ClearOnDone    bool          // Clear screen when done
	Compact        bool          // Use compact display
	NoColor        bool          // Disable colors
}

// DefaultOptions returns sensible default options
func DefaultOptions() TrackerOptions {
	return TrackerOptions{
		ShowElapsed:    true,
		ShowDuration:   true,
		ShowProgress:   true,
		AnimationSpeed: 100 * time.Millisecond,
		ClearOnDone:    false,
		Compact:        false,
		NoColor:        false,
	}
}

// NewTracker creates a new progress tracker
func NewTracker(title string, opts ...TrackerOptions) *Tracker {
	options := DefaultOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	return &Tracker{
		title:     title,
		steps:     make(map[string]*Step),
		stepOrder: make([]string, 0),
		animation: animationState{lastUpdate: time.Now()},
		done:      make(chan struct{}),
		options:   options,
	}
}

// AddStep adds a new step to track
func (t *Tracker) AddStep(id, name string) *Step {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.steps[id]; !exists {
		t.stepOrder = append(t.stepOrder, id)
	}

	step := &Step{
		ID:        id,
		Name:      name,
		Status:    StatusPending,
		StartTime: time.Now(),
		Active:    false,
	}

	t.steps[id] = step
	return step
}

// GetStep returns a step by ID
func (t *Tracker) GetStep(id string) (*Step, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	step, exists := t.steps[id]
	return step, exists
}

// StartStep marks a step as running
func (t *Tracker) StartStep(id string, message ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if step, exists := t.steps[id]; exists {
		step.Status = StatusRunning
		step.StartTime = time.Now()
		step.Active = true
		if len(message) > 0 {
			step.Message = message[0]
		}
	}
}

// UpdateStep updates a step's message and optionally progress
func (t *Tracker) UpdateStep(id, message string, progress ...float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if step, exists := t.steps[id]; exists {
		step.Message = message
		if len(progress) > 0 {
			step.Progress = progress[0]
		}
	}
}

// CompleteStep marks a step as completed
func (t *Tracker) CompleteStep(id string, message ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if step, exists := t.steps[id]; exists {
		step.Status = StatusCompleted
		step.EndTime = time.Now()
		step.Active = false
		if len(message) > 0 {
			step.Message = message[0]
		}
	}
}

// FailStep marks a step as failed
func (t *Tracker) FailStep(id, errorMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if step, exists := t.steps[id]; exists {
		step.Status = StatusFailed
		step.Error = errorMsg
		step.EndTime = time.Now()
		step.Active = false
	}
}

// SkipStep marks a step as skipped
func (t *Tracker) SkipStep(id string, reason ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if step, exists := t.steps[id]; exists {
		step.Status = StatusSkipped
		step.EndTime = time.Now()
		step.Active = false
		if len(reason) > 0 {
			step.Message = reason[0]
		}
	}
}

// Start begins the animation loop
func (t *Tracker) Start() {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.running = true
	t.mu.Unlock()

	go t.animationLoop()
}

// Stop stops the animation and shows final state
func (t *Tracker) Stop() {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return
	}
	t.running = false
	t.mu.Unlock()

	close(t.done)

	if t.options.ClearOnDone {
		t.renderFinalState()
	} else {
		t.render(true) // Render final state without animation
	}
}

// animationLoop runs the animation updates
func (t *Tracker) animationLoop() {
	ticker := time.NewTicker(t.options.AnimationSpeed)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			return
		case <-ticker.C:
			t.updateAnimation()
			t.render(false)
		}
	}
}

// updateAnimation updates the animation frame
func (t *Tracker) updateAnimation() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if now.Sub(t.animation.lastUpdate) >= t.options.AnimationSpeed {
		t.animation.frame++
		t.animation.lastUpdate = now
	}
}

// render displays the current state
func (t *Tracker) render(final bool) {
	if !final {
		// Clear screen and move cursor to top for live updates
		fmt.Print("\033[H\033[J")
	}

	// Title
	titleColor := t.color(ColorBlue)
	fmt.Printf("%s%s%s\n", titleColor, t.title, t.colorReset())

	if !t.options.Compact {
		fmt.Println(strings.Repeat("─", 50))
	}

	// Render steps
	t.mu.RLock()
	for _, stepID := range t.stepOrder {
		step := t.steps[stepID]
		t.renderStep(step, final)
	}
	t.mu.RUnlock()

	fmt.Println()
}

// renderStep renders a single step
func (t *Tracker) renderStep(step *Step, final bool) {
	step.mu.RLock()
	defer step.mu.RUnlock()

	icon, color := t.getStepIcon(step, final)

	// Step name with icon
	fmt.Printf("%s %s%s%s", icon, color, step.Name, t.colorReset())

	// Show elapsed time for running steps
	if t.options.ShowElapsed && step.Status == StatusRunning && step.Active && !final {
		elapsed := time.Since(step.StartTime).Truncate(time.Second)
		fmt.Printf(" %s(%s)%s", t.color(ColorGray), elapsed, t.colorReset())
	}

	// Show duration for completed steps
	if t.options.ShowDuration && step.Status == StatusCompleted && !step.EndTime.IsZero() {
		duration := step.EndTime.Sub(step.StartTime).Truncate(time.Millisecond)
		fmt.Printf(" %s(%s)%s", t.color(ColorGreen), duration, t.colorReset())
	}

	fmt.Println()

	// Show message
	if step.Message != "" {
		indent := "  "
		message := step.Message

		// Add animated dots for running steps
		if step.Status == StatusRunning && step.Active && !final {
			dots := DotsChars[t.animation.frame%len(DotsChars)]
			message = message + dots
		}

		fmt.Printf("%s%s\n", indent, message)
	}

	// Show progress bar if available
	if t.options.ShowProgress && step.Progress > 0 && step.Status == StatusRunning {
		t.renderProgressBar(step.Progress)
	}

	// Show error if present
	if step.Error != "" {
		fmt.Printf("  %sError: %s%s\n", t.color(ColorRed), step.Error, t.colorReset())
	}
}

// getStepIcon returns the appropriate icon and color for a step
func (t *Tracker) getStepIcon(step *Step, final bool) (string, string) {
	switch step.Status {
	case StatusPending:
		return t.colorize("⏳", ColorYellow), t.color(ColorYellow)
	case StatusRunning:
		if step.Active && !final {
			char := SpinnerChars[t.animation.frame%len(SpinnerChars)]
			return t.colorize(char, ColorCyan), t.color(ColorCyan)
		}
		return t.colorize("⚙️", ColorCyan), t.color(ColorCyan)
	case StatusCompleted:
		return t.colorize("✅", ColorGreen), t.color(ColorGreen)
	case StatusFailed:
		return t.colorize("❌", ColorRed), t.color(ColorRed)
	case StatusSkipped:
		return t.colorize("⏭️", ColorGray), t.color(ColorGray)
	default:
		return t.colorize("⏳", ColorYellow), t.color(ColorYellow)
	}
}

// renderProgressBar renders a progress bar
func (t *Tracker) renderProgressBar(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	width := 30
	filled := int(progress * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	percentage := int(progress * 100)

	fmt.Printf("  %s[%s%s%s] %d%%\n",
		t.color(ColorCyan),
		t.color(ColorGreen),
		bar,
		t.colorReset(),
		percentage)
}

// renderFinalState shows the final state
func (t *Tracker) renderFinalState() {
	fmt.Print("\033[H\033[J")
	fmt.Printf("%s%s - Complete%s\n", t.color(ColorGreen), t.title, t.colorReset())
	fmt.Println(strings.Repeat("─", 50))

	t.mu.RLock()
	for _, stepID := range t.stepOrder {
		step := t.steps[stepID]
		t.renderStep(step, true)
	}
	t.mu.RUnlock()

	fmt.Println()
}

// color returns color code if colors are enabled
func (t *Tracker) color(color string) string {
	if t.options.NoColor {
		return ""
	}
	return color
}

// colorReset returns reset code if colors are enabled
func (t *Tracker) colorReset() string {
	if t.options.NoColor {
		return ""
	}
	return ColorReset
}

// colorize wraps text in color if colors are enabled
func (t *Tracker) colorize(text, color string) string {
	if t.options.NoColor {
		return text
	}
	return color + text + ColorReset
}

func BuildProgress(target string) *Tracker {
	opts := DefaultOptions()
	opts.ClearOnDone = false // Don't clear screen when done
	opts.ShowElapsed = true
	opts.ShowDuration = true

	tracker := NewTracker(fmt.Sprintf("Building %s", target), opts)
	tracker.AddStep("prepare", "Preparing build environment")
	tracker.AddStep("dependencies", "Installing dependencies")
	tracker.AddStep("compile", "Compiling")
	tracker.AddStep("package", "Packaging")
	tracker.AddStep("verify", "Verifying build")
	tracker.Start()
	return tracker
}

func DeployProgress() *Tracker {
	opts := DefaultOptions()
	opts.ShowElapsed = true
	opts.ShowDuration = true

	tracker := NewTracker("Deployment Progress", opts)
	tracker.AddStep("pending", "Version queued")
	tracker.AddStep("building", "Building deployment")
	tracker.AddStep("deploying", "Deploying to infrastructure")
	tracker.AddStep("active", "Activation complete")
	tracker.Start()
	return tracker
}
