package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// StreamingOutput provides a fixed-height scrolling view of command output
type StreamingOutput struct {
	title      string
	lines      []string
	maxLines   int
	mu         sync.Mutex
	done       chan bool
	lastUpdate time.Time
}

// NewStreamingOutput creates a new streaming output display
func NewStreamingOutput(title string, maxLines int) *StreamingOutput {
	if maxLines <= 0 {
		maxLines = 5
	}
	return &StreamingOutput{
		title:      title,
		lines:      make([]string, 0, maxLines),
		maxLines:   maxLines,
		done:       make(chan bool, 1),
		mu:         sync.Mutex{},
		lastUpdate: time.Now(),
	}
}

// Start begins the display update loop
func (s *StreamingOutput) Start() {
	// Clear the lines we'll be using
	for i := 0; i < s.maxLines+2; i++ {
		fmt.Println()
	}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinIdx := 0

		for {
			select {
			case <-s.done:
				s.render("✓", true)
				return
			case <-ticker.C:
				s.render(spinners[spinIdx%len(spinners)], false)
				spinIdx++
			}
		}
	}()
}

// Stream reads from the reader and displays lines in the viewport
func (s *StreamingOutput) Stream(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		// Clean up the line
		line = strings.TrimSpace(line)
		if line != "" {
			s.AddLine(line)
		}
	}
}

// AddLine adds a line to the display
func (s *StreamingOutput) AddLine(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Truncate long lines
	if len(line) > 80 {
		line = line[:77] + "..."
	}

	// Add to buffer
	s.lines = append(s.lines, line)

	// Keep only the last N lines
	if len(s.lines) > s.maxLines {
		s.lines = s.lines[len(s.lines)-s.maxLines:]
	}

	s.lastUpdate = time.Now()
}

// Stop stops the display and shows final state
func (s *StreamingOutput) Stop() {
	close(s.done)
	time.Sleep(150 * time.Millisecond) // Let final render complete
}

// render updates the display
func (s *StreamingOutput) render(spinner string, final bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Move cursor up to start of our display area
	fmt.Printf("\033[%dA", s.maxLines+2)

	// Title line
	if final {
		fmt.Printf("%s %s\033[K\n", spinner, s.title)
	} else {
		fmt.Printf("%s %s\033[K\n", spinner, s.title)
	}

	// Separator
	fmt.Printf("   ├─────────────────────────────────────────────────────────────────────────\033[K\n")

	// Content lines
	for i := 0; i < s.maxLines; i++ {
		if i < len(s.lines) {
			fmt.Printf("   │ %s\033[K\n", s.lines[i])
		} else {
			fmt.Printf("   │\033[K\n")
		}
	}

	// Bottom border
	if final {
		fmt.Printf("   └─────────────────────────────────────────────────────────────────────────\033[K\n")
	}
}

// RunCommandWithOutput runs a command and displays its output in a streaming viewport
func RunCommandWithOutput(title string, command io.Reader) error {
	output := NewStreamingOutput(title, 5)
	output.Start()
	output.Stream(command)
	output.Stop()
	return nil
}
