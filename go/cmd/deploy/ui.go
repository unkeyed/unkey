package deploy

import (
	"fmt"
	"sync"
	"time"
)

// Colors
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
)

// Symbols
const (
	SymbolTick   = "✔"
	SymbolCross  = "✘"
	SymbolBullet = "●"
	SymbolArrow  = "=>"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type UI struct {
	mu           sync.Mutex
	spinning     bool
	currentStep  string
	stepSpinning bool
}

func NewUI() *UI {
	return &UI{
		mu:           sync.Mutex{},
		spinning:     false,
		currentStep:  "",
		stepSpinning: false,
	}
}

func (ui *UI) Print(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("%s%s%s %s\n", ColorYellow, SymbolBullet, ColorReset, message)
}

func (ui *UI) PrintSuccess(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("%s%s%s %s\n", ColorGreen, SymbolTick, ColorReset, message)
}

func (ui *UI) PrintError(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("%s%s%s %s\n", ColorRed, SymbolCross, ColorReset, message)
}

func (ui *UI) PrintErrorDetails(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("  %s%s%s %s\n", ColorRed, SymbolArrow, ColorReset, message)
}

func (ui *UI) PrintStepSuccess(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("  %s%s%s %s\n", ColorGreen, SymbolTick, ColorReset, message)
}

func (ui *UI) PrintStepError(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("  %s%s%s %s\n", ColorRed, SymbolCross, ColorReset, message)
}

func (ui *UI) spinnerLoop(prefix string, messageGetter func() string, isActive func() bool) {
	go func() {
		frame := 0
		for {
			ui.mu.Lock()
			if !isActive() {
				ui.mu.Unlock()
				return
			}
			message := messageGetter()
			fmt.Printf("\r%s%s %s", prefix, spinnerChars[frame%len(spinnerChars)], message)
			ui.mu.Unlock()
			frame++
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (ui *UI) StartSpinner(message string) {
	ui.mu.Lock()
	if ui.spinning {
		ui.mu.Unlock()
		return
	}
	ui.spinning = true
	spinnerMessage := message
	ui.mu.Unlock()

	ui.spinnerLoop("", func() string { return spinnerMessage }, func() bool { return ui.spinning })
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
		fmt.Printf("%s%s%s %s\n", ColorGreen, SymbolTick, ColorReset, finalMessage)
	} else {
		fmt.Printf("%s%s%s %s\n", ColorRed, SymbolCross, ColorReset, finalMessage)
	}
}

// Step spinner methods - indented with 2 spaces to show as sub-steps
func (ui *UI) StartStepSpinner(message string) {
	ui.mu.Lock()
	if ui.stepSpinning {
		fmt.Print("\r\033[K")
	}
	ui.currentStep = message
	ui.stepSpinning = true
	ui.mu.Unlock()
	ui.spinnerLoop("  ", func() string { return ui.currentStep }, func() bool { return ui.stepSpinning })
}

func (ui *UI) CompleteStepAndStartNext(completedMessage, nextMessage string) {
	ui.mu.Lock()
	// Stop current spinner and show completion
	if ui.stepSpinning {
		ui.stepSpinning = false
		fmt.Print("\r\033[K")
		fmt.Printf("  %s%s%s %s\n", ColorGreen, SymbolTick, ColorReset, completedMessage)
	}

	// Start next step if provided
	if nextMessage != "" {
		ui.currentStep = nextMessage
		ui.stepSpinning = true
		ui.mu.Unlock()

		ui.spinnerLoop("  ", func() string { return ui.currentStep }, func() bool { return ui.stepSpinning })
	} else {
		ui.mu.Unlock()
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
		fmt.Printf("  %s%s%s %s\n", ColorGreen, SymbolTick, ColorReset, message)
	} else {
		fmt.Printf("  %s%s%s %s\n", ColorRed, SymbolCross, ColorReset, message)
	}
}
