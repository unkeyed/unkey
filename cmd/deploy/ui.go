package deploy

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Colors and Symbols
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"

	SymbolTick   = "✔"
	SymbolCross  = "✘"
	SymbolBullet = "●"
	SymbolArrow  = "=>"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type UI struct {
	mu            sync.Mutex
	spinnerCancel context.CancelFunc
}

func NewUI() *UI {
	return &UI{
		mu:            sync.Mutex{},
		spinnerCancel: nil,
	}
}

func (ui *UI) print(color, symbol, indent, message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("%s%s%s%s %s\n", indent, color, symbol, ColorReset, message)
}

func (ui *UI) Print(message string)             { ui.print(ColorYellow, SymbolBullet, "", message) }
func (ui *UI) PrintSuccess(message string)      { ui.print(ColorGreen, SymbolTick, "", message) }
func (ui *UI) PrintError(message string)        { ui.print(ColorRed, SymbolCross, "", message) }
func (ui *UI) PrintErrorDetails(message string) { ui.print(ColorRed, SymbolArrow, "  ", message) }
func (ui *UI) PrintStepSuccess(message string)  { ui.print(ColorGreen, SymbolTick, "  ", message) }
func (ui *UI) PrintStepError(message string)    { ui.print(ColorRed, SymbolCross, "  ", message) }

// StartSpinner starts a spinner with the given message and indentation
func (ui *UI) StartSpinner(message string) {
	ui.startSpinner(message, "")
}

func (ui *UI) StartStepSpinner(message string) {
	ui.startSpinner(message, "  ")
}

func (ui *UI) startSpinner(message, indent string) {
	ui.mu.Lock()
	// Stop any existing spinner first
	if ui.spinnerCancel != nil {
		ui.spinnerCancel()
		fmt.Print("\r\033[K")
	}

	ctx, cancel := context.WithCancel(context.Background())
	ui.spinnerCancel = cancel
	ui.mu.Unlock()

	go func() {
		frame := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Printf("\r%s%s %s", indent, spinnerChars[frame%len(spinnerChars)], message)
				frame++
			}
		}
	}()
}

// StopSpinner stops the current spinner and shows final message
func (ui *UI) StopSpinner(finalMessage string, success bool) {
	ui.stopSpinner(finalMessage, success, "")
}

func (ui *UI) CompleteCurrentStep(message string, success bool) {
	ui.stopSpinner(message, success, "  ")
}

func (ui *UI) stopSpinner(message string, success bool, indent string) {
	ui.mu.Lock()
	if ui.spinnerCancel != nil {
		ui.spinnerCancel()
		ui.spinnerCancel = nil
	}
	ui.mu.Unlock()

	// Clear spinner line
	fmt.Print("\r\033[K")

	// Show final message
	if success {
		ui.print(ColorGreen, SymbolTick, indent, message)
	} else {
		ui.print(ColorRed, SymbolCross, indent, message)
	}
}

// CompleteStepAndStartNext completes current step and starts next one
func (ui *UI) CompleteStepAndStartNext(completedMessage, nextMessage string) {
	ui.CompleteCurrentStep(completedMessage, true)
	if nextMessage != "" {
		ui.StartStepSpinner(nextMessage)
	}
}
