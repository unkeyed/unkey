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
	mu           sync.Mutex
	spinning     bool
	currentStep  string
	stepSpinning bool
}

func NewUI() *UI {
	return &UI{}
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

// Step spinner methods - indented with 2 spaces to show as sub-steps
func (ui *UI) StartStepSpinner(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.stepSpinning {
		fmt.Print("\r\033[K")
	}

	ui.currentStep = message
	ui.stepSpinning = true

	go func() {
		frame := 0
		for {
			ui.mu.Lock()
			if !ui.stepSpinning {
				ui.mu.Unlock()
				return
			}
			fmt.Printf("\r  %s %s", spinnerChars[frame%len(spinnerChars)], ui.currentStep)
			ui.mu.Unlock()
			frame++
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (ui *UI) CompleteStepAndStartNext(completedMessage, nextMessage string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Stop current spinner and show completion
	if ui.stepSpinning {
		ui.stepSpinning = false
		fmt.Print("\r\033[K")
		fmt.Printf("  %s✓%s %s\n", ColorGreen, ColorReset, completedMessage)
	}

	// Start next step if provided
	if nextMessage != "" {
		ui.currentStep = nextMessage
		ui.stepSpinning = true

		go func() {
			frame := 0
			for {
				ui.mu.Lock()
				if !ui.stepSpinning {
					ui.mu.Unlock()
					return
				}
				fmt.Printf("\r  %s %s", spinnerChars[frame%len(spinnerChars)], ui.currentStep)
				ui.mu.Unlock()
				frame++
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}
}

func (ui *UI) CompleteCurrentStep(message string, success bool) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if !ui.stepSpinning {
		return
	}

	ui.stepSpinning = false
	fmt.Print("\r\033[K")

	if success {
		fmt.Printf("  %s✓%s %s\n", ColorGreen, ColorReset, message)
	} else {
		fmt.Printf("  %s✗%s %s\n", ColorRed, ColorReset, message)
	}
}
