package deploy

import (
	"fmt"
	"sync"
	"time"
)

// Color constants
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type UI struct {
	mu       sync.Mutex
	spinning bool
	done     chan struct{}
}

func NewUI() *UI {
	return &UI{
		done: make(chan struct{}),
	}
}

func (ui *UI) Print(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("%s•%s %s\n", ColorYellow, ColorReset, message)
}

func (ui *UI) PrintSuccess(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("%s✓%s %s\n", ColorGreen, ColorReset, message)
}

func (ui *UI) PrintError(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("%s✗%s %s\n", ColorRed, ColorReset, message)
}

func (ui *UI) PrintErrorDetails(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("  %s->%s %s\n", ColorRed, ColorReset, message)
}

func (ui *UI) PrintStepSuccess(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("  %s✓%s %s\n", ColorGreen, ColorReset, message)
}

func (ui *UI) PrintStepError(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("  %s✗%s %s\n", ColorRed, ColorReset, message)
}

func (ui *UI) StartSpinner(message string) {
	ui.mu.Lock()
	if ui.spinning {
		ui.mu.Unlock()
		return
	}
	ui.spinning = true
	ui.mu.Unlock()

	go func() {
		frame := 0
		for {
			select {
			case <-ui.done:
				return
			default:
				ui.mu.Lock()
				if !ui.spinning {
					ui.mu.Unlock()
					return
				}
				fmt.Printf("\r%s %s", spinnerChars[frame%len(spinnerChars)], message)
				ui.mu.Unlock()
				frame++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (ui *UI) StopSpinner(finalMessage string, success bool) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if !ui.spinning {
		return
	}

	ui.spinning = false
	fmt.Print("\r\033[K")

	if success {
		fmt.Printf("%s✓%s %s\n", ColorGreen, ColorReset, finalMessage)
	} else {
		fmt.Printf("%s✗%s %s\n", ColorRed, ColorReset, finalMessage)
	}
}
