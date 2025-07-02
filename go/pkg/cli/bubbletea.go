package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// Messages
type outputLine struct {
	line string
}

type commandComplete struct {
	err error
}

// Model for the streaming output view
type streamingModel struct {
	title       string
	viewport    viewport.Model
	lines       []string
	spinner     spinner.Model
	done        bool
	err         error
	width       int
	height      int
	showSpinner bool
}

// NewStreamingModel creates a new streaming output model
func NewStreamingModel(title string) streamingModel {
	vp := viewport.New(80, 8)
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	return streamingModel{
		title:       title,
		viewport:    vp,
		spinner:     s,
		lines:       make([]string, 0),
		showSpinner: true,
		width:       80,
		height:      12,
		done:        false,
		err:         nil,
	}
}

func (m streamingModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m streamingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = 12 // Fixed height for output area
		m.viewport.Width = m.width - 4
		m.viewport.Height = 8

	case outputLine:
		// Add new line and update viewport
		m.lines = append(m.lines, msg.line)
		content := strings.Join(m.lines, "\n")
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()

	case commandComplete:
		m.done = true
		m.err = msg.err
		m.showSpinner = false
		return m, tea.Quit

	case spinner.TickMsg:
		if !m.done && m.showSpinner {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m streamingModel) View() string {
	var title string
	if m.done {
		if m.err != nil {
			title = errorStyle.Render("✗ ") + m.title + errorStyle.Render(" (failed)")
		} else {
			title = successStyle.Render("✓ ") + m.title + successStyle.Render(" (completed)")
		}
	} else {
		title = m.spinner.View() + " " + titleStyle.Render(m.title)
	}

	// Create the box with viewport content
	box := boxStyle.Width(m.width - 2).Render(m.viewport.View())

	// Add scroll indicator if needed
	scrollInfo := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		scrollPercent := int(float64(m.viewport.YOffset+m.viewport.Height) / float64(m.viewport.TotalLineCount()) * 100)
		if scrollPercent > 100 {
			scrollPercent = 100
		}
		scrollInfo = dimStyle.Render(fmt.Sprintf(" ↓ %d%%", scrollPercent))
	}

	return fmt.Sprintf("%s\n%s%s\n", title, box, scrollInfo)
}

// StreamCommand runs a command and displays output in a nice Bubble Tea UI
func StreamCommand(title string, cmdReader io.Reader, cmdDone <-chan error) error {
	p := tea.NewProgram(NewStreamingModel(title))

	// Start reading command output in a goroutine
	go func() {
		defer func() {
			// Always wait for command completion and send the result
			cmdErr := <-cmdDone
			p.Send(commandComplete{err: cmdErr})
		}()

		scanner := bufio.NewScanner(cmdReader)
		scanner.Split(bufio.ScanLines)

		// Read lines as they come
		for scanner.Scan() {
			line := scanner.Text()
			// Clean up docker build output
			line = cleanDockerOutput(line)
			if line != "" {
				p.Send(outputLine{line: line})
			}
		}

		// Check for scanner errors
		if err := scanner.Err(); err != nil {
			p.Send(outputLine{line: fmt.Sprintf("Scanner error: %v", err)})
		}
	}()

	// Run the Bubble Tea program
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}

// cleanDockerOutput cleans up Docker build output for better display
func cleanDockerOutput(line string) string {
	line = strings.TrimSpace(line)

	// Remove ANSI escape codes
	line = stripANSI(line)

	// Skip empty lines
	if line == "" {
		return ""
	}

	// Skip certain Docker output that's just noise
	if strings.HasPrefix(line, "#") && len(line) < 5 {
		return ""
	}

	// Shorten long lines
	if len(line) > 76 {
		line = line[:73] + "..."
	}

	return line
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	// Simple ANSI stripping - you might want a more robust solution
	var result strings.Builder
	inEscape := false

	for _, ch := range str {
		if ch == '\033' {
			inEscape = true
		} else if inEscape && ch == 'm' {
			inEscape = false
		} else if !inEscape {
			result.WriteRune(ch)
		}
	}

	return result.String()
}
