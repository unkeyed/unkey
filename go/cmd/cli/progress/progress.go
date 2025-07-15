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
	SpinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	DotsChars    = []string{"", ".", "..", "..."}
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
	metadata  map[string]any
	mu        sync.RWMutex
}

// SetMetadata sets custom metadata for the step
func (s *Step) SetMetadata(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.metadata == nil {
		s.metadata = make(map[string]any)
	}
	s.metadata[key] = value
}

// GetMetadata gets custom metadata for the step
func (s *Step) GetMetadata(key string) (any, bool) {
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
	title       string
	steps       map[string]*Step
	stepOrder   []string
	animation   animationState
	done        chan struct{}
	running     bool
	mu          sync.RWMutex
	options     TrackerOptions
	renderState renderState
	firstRender bool
}

type animationState struct {
	frame      int
	lastUpdate time.Time
}

type renderState struct {
	linesRendered int
	lastContent   []string
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
		title:       title,
		steps:       make(map[string]*Step),
		stepOrder:   make([]string, 0),
		animation:   animationState{lastUpdate: time.Now()},
		done:        make(chan struct{}),
		options:     options,
		firstRender: true,
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

	// Do initial render immediately to avoid race conditions
	t.render(false)

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

	// Wait a brief moment for the animation loop to finish
	time.Sleep(50 * time.Millisecond)

	// Ensure final state is rendered
	t.render(true)
}

// animationLoop runs the animation updates
func (t *Tracker) animationLoop() {
	ticker := time.NewTicker(t.options.AnimationSpeed)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			// Render final state before exiting
			if t.options.ClearOnDone {
				t.renderFinalState()
			} else {
				t.render(true) // Final render without animation
			}
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

// render displays the current state with minimal layout shift
func (t *Tracker) render(final bool) {
	// Build the complete content first
	content := t.buildContent(final)

	if t.firstRender {
		// First render - just print everything
		for _, line := range content {
			fmt.Println(line)
		}
		t.firstRender = false
		t.renderState.linesRendered = len(content)
		t.renderState.lastContent = make([]string, len(content))
		copy(t.renderState.lastContent, content)
		return
	}

	// Update only changed lines
	t.updateChangedLines(content)

	// Store current content for next comparison
	t.renderState.lastContent = make([]string, len(content))
	copy(t.renderState.lastContent, content)
}

// buildContent builds the complete content as slice of lines
func (t *Tracker) buildContent(final bool) []string {
	var content []string

	// Title
	titleColor := t.color(ColorBlue)
	content = append(content, fmt.Sprintf("%s%s%s", titleColor, t.title, t.colorReset()))

	if !t.options.Compact {
		content = append(content, strings.Repeat("─", 50))
	}

	// Build step content
	t.mu.RLock()
	for _, stepID := range t.stepOrder {
		step := t.steps[stepID]
		stepLines := t.buildStepContent(step, final)
		content = append(content, stepLines...)
	}
	t.mu.RUnlock()

	content = append(content, "") // Empty line at the end

	return content
}

// buildStepContent builds content for a single step
func (t *Tracker) buildStepContent(step *Step, final bool) []string {
	var lines []string

	step.mu.RLock()
	defer step.mu.RUnlock()

	icon, color := t.getStepIcon(step, final)

	// Step name with icon
	stepLine := fmt.Sprintf("%s %s%s%s", icon, color, step.Name, t.colorReset())

	// Show elapsed time for running steps
	if t.options.ShowElapsed && step.Status == StatusRunning && step.Active && !final {
		elapsed := time.Since(step.StartTime).Truncate(time.Second)
		stepLine += fmt.Sprintf(" %s(%s)%s", t.color(ColorGray), elapsed, t.colorReset())
	}

	// Show duration for completed steps
	if t.options.ShowDuration && step.Status == StatusCompleted && !step.EndTime.IsZero() {
		duration := step.EndTime.Sub(step.StartTime).Truncate(time.Millisecond)
		stepLine += fmt.Sprintf(" %s(%s)%s", t.color(ColorGreen), duration, t.colorReset())
	}

	lines = append(lines, stepLine)

	// Show message
	if step.Message != "" {
		indent := "  "
		message := step.Message

		// Add animated dots for running steps
		if step.Status == StatusRunning && step.Active && !final {
			dots := DotsChars[t.animation.frame%len(DotsChars)]
			message = message + dots
		}

		lines = append(lines, fmt.Sprintf("%s%s", indent, message))
	}

	// Show progress bar if available
	if t.options.ShowProgress && step.Progress > 0 && step.Status == StatusRunning {
		progressLine := t.buildProgressBar(step.Progress)
		lines = append(lines, progressLine)
	}

	// Show error if present
	if step.Error != "" {
		errorLine := fmt.Sprintf("  %s -> Error: %s%s", t.color(ColorRed), step.Error, t.colorReset())
		lines = append(lines, errorLine)
	}

	return lines
}

// updateChangedLines updates only the lines that have changed
func (t *Tracker) updateChangedLines(newContent []string) {
	maxLines := max(len(t.renderState.lastContent), len(newContent))

	for i := range maxLines {
		var newLine, oldLine string

		if i < len(newContent) {
			newLine = newContent[i]
		}
		if i < len(t.renderState.lastContent) {
			oldLine = t.renderState.lastContent[i]
		}

		if newLine != oldLine {
			// Move cursor to the line and clear it
			fmt.Printf("\033[%d;1H\033[K%s", i+1, newLine)
		}
	}

	// If we have fewer lines now, clear the remaining ones
	if len(newContent) < len(t.renderState.lastContent) {
		for i := len(newContent); i < len(t.renderState.lastContent); i++ {
			fmt.Printf("\033[%d;1H\033[K", i+1)
		}
	}

	t.renderState.linesRendered = len(newContent)
}

// buildProgressBar builds a progress bar string
func (t *Tracker) buildProgressBar(progress float64) string {
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

	return fmt.Sprintf("  %s[%s%s%s] %d%%",
		t.color(ColorCyan),
		t.color(ColorGreen),
		bar,
		t.colorReset(),
		percentage)
}

// getStepIcon returns the appropriate icon and color for a step
func (t *Tracker) getStepIcon(step *Step, final bool) (string, string) {
	switch step.Status {
	case StatusPending:
		return t.colorize("○", ColorYellow), t.color(ColorYellow)
	case StatusRunning:
		if step.Active && !final {
			char := SpinnerChars[t.animation.frame%len(SpinnerChars)]
			return t.colorize(char, ColorCyan), t.color(ColorCyan)
		}
		return t.colorize("●", ColorCyan), t.color(ColorCyan)
	case StatusCompleted:
		return t.colorize("✓", ColorGreen), t.color(ColorGreen)
	case StatusFailed:
		return t.colorize("✗", ColorRed), t.color(ColorRed)
	case StatusSkipped:
		return t.colorize("⊘", ColorGray), t.color(ColorGray)
	default:
		return t.colorize("○", ColorYellow), t.color(ColorYellow)
	}
}

// renderFinalState shows the final state
func (t *Tracker) renderFinalState() {
	fmt.Print("\033[H\033[J")
	fmt.Printf("%s%s - Complete%s\n", t.color(ColorGreen), t.title, t.colorReset())
	fmt.Println(strings.Repeat("─", 50))

	t.mu.RLock()
	for _, stepID := range t.stepOrder {
		step := t.steps[stepID]
		stepLines := t.buildStepContent(step, true)
		for _, line := range stepLines {
			fmt.Println(line)
		}
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
