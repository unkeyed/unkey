package cli

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// StepState represents the state of a deployment step
type StepState int

const (
	StepStatePending StepState = iota
	StepStateRunning
	StepStateComplete
	StepStateFailed
)

// DeploymentDisplay manages a live-updating deployment output
type DeploymentDisplay struct {
	mu        sync.RWMutex
	sections  []*Section
	header    string
	startTime time.Time
}

// Section represents a major deployment section (Building, Publishing, etc.)
type Section struct {
	Name      string
	State     StepState
	StartTime time.Time
	EndTime   time.Time
	Steps     []*Step
}

// Step represents a sub-step within a section
type Step struct {
	Name    string
	State   StepState
	Details string
	Indent  int
}

// Styles for the display
var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))

	pendingIcon  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("○")
	runningIcon  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("○")
	completeIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	failedIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗")

	sectionNameStyle = lipgloss.NewStyle().Bold(true)
	stepNameStyle    = lipgloss.NewStyle()
	detailsStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	durationStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// NewDeploymentDisplay creates a new deployment display
func NewDeploymentDisplay(version, path string) *DeploymentDisplay {
	header := fmt.Sprintf("DEPLOY  %s  %s", version, path)

	return &DeploymentDisplay{
		mu:        sync.RWMutex{},
		header:    header,
		startTime: time.Now(),
		sections:  make([]*Section, 0),
	}
}

// AddSection adds a new section to the deployment
func (d *DeploymentDisplay) AddSection(name string) *Section {
	d.mu.Lock()
	defer d.mu.Unlock()

	section := &Section{
		Name:      name,
		State:     StepStatePending,
		StartTime: time.Time{},
		EndTime:   time.Time{},
		Steps:     make([]*Step, 0),
	}

	d.sections = append(d.sections, section)
	return section
}

// StartSection marks a section as running
func (d *DeploymentDisplay) StartSection(section *Section) {
	d.mu.Lock()
	defer d.mu.Unlock()

	section.State = StepStateRunning
	section.StartTime = time.Now()
	d.render()
}

// CompleteSection marks a section as complete
func (d *DeploymentDisplay) CompleteSection(section *Section) {
	d.mu.Lock()
	defer d.mu.Unlock()

	section.State = StepStateComplete
	section.EndTime = time.Now()
	d.render()
}

// FailSection marks a section as failed
func (d *DeploymentDisplay) FailSection(section *Section) {
	d.mu.Lock()
	defer d.mu.Unlock()

	section.State = StepStateFailed
	section.EndTime = time.Now()
	d.render()
}

// AddStep adds a step to a section
func (d *DeploymentDisplay) AddStep(section *Section, name string, indent int) *Step {
	d.mu.Lock()
	defer d.mu.Unlock()

	step := &Step{
		Name:    name,
		State:   StepStatePending,
		Details: "",
		Indent:  indent,
	}

	section.Steps = append(section.Steps, step)
	return step
}

// CompleteStep marks a step as complete
func (d *DeploymentDisplay) CompleteStep(step *Step) {
	d.mu.Lock()
	defer d.mu.Unlock()

	step.State = StepStateComplete
	d.render()
}

// UpdateStepDetails updates the details for a step
func (d *DeploymentDisplay) UpdateStepDetails(step *Step, details string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	step.Details = details
	d.render()
}

// render displays the current state (called with mutex held)
func (d *DeploymentDisplay) render() {
	// Clear screen and move to top
	fmt.Print("\033[2J\033[H")

	// Show header
	fmt.Println(headerStyle.Render(d.header))
	fmt.Println()

	// Show each section
	for _, section := range d.sections {
		d.renderSection(section)
		fmt.Println()
	}
}

// renderSection renders a single section
func (d *DeploymentDisplay) renderSection(section *Section) {
	icon := d.getStateIcon(section.State)
	name := sectionNameStyle.Render(section.Name)

	line := fmt.Sprintf("%s %s", icon, name)

	// Add duration if section has started
	if !section.StartTime.IsZero() {
		var duration time.Duration
		if section.State == StepStateComplete || section.State == StepStateFailed {
			duration = section.EndTime.Sub(section.StartTime)
		} else {
			duration = time.Since(section.StartTime)
		}

		durationStr := fmt.Sprintf("(%.1fs)", duration.Seconds())
		line += " " + durationStyle.Render(durationStr)
	}

	fmt.Println(line)

	// Render steps
	for _, step := range section.Steps {
		d.renderStep(step)
	}
}

// renderStep renders a single step
func (d *DeploymentDisplay) renderStep(step *Step) {
	indent := strings.Repeat("  ", step.Indent+1)
	icon := d.getStateIcon(step.State)
	name := stepNameStyle.Render(step.Name)

	line := fmt.Sprintf("%s%s %s", indent, icon, name)

	if step.Details != "" {
		line += ": " + detailsStyle.Render(step.Details)
	}

	fmt.Println(line)
}

// getStateIcon returns the appropriate icon for a state
func (d *DeploymentDisplay) getStateIcon(state StepState) string {
	switch state {
	case StepStatePending:
		return pendingIcon
	case StepStateRunning:
		return runningIcon
	case StepStateComplete:
		return completeIcon
	case StepStateFailed:
		return failedIcon
	default:
		return pendingIcon
	}
}

// Show displays the final state
func (d *DeploymentDisplay) Show() {
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.render()
}
