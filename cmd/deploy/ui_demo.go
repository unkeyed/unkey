//go:build ignore

// This file is a standalone demo to visualize the UI components in isolation.
// The build tags above prevent it from being compiled with the deploy package.
//
// Run with: go run cmd/deploy/ui_demo.go

package main

import (
	"time"

	"github.com/unkeyed/unkey/cmd/deploy"
)

func main() {
	ui := deploy.NewUI()

	// Test basic prints
	ui.Print("Starting deployment process...")
	time.Sleep(500 * time.Millisecond)

	ui.PrintSuccess("Environment validated")
	time.Sleep(500 * time.Millisecond)

	ui.PrintError("Failed to connect to database")
	time.Sleep(500 * time.Millisecond)

	ui.PrintErrorDetails("Connection timeout after 30s")
	time.Sleep(500 * time.Millisecond)

	// Test spinner
	ui.StartSpinner("Deploying service...")
	time.Sleep(3 * time.Second)
	ui.StopSpinner("Service deployed successfully", true)

	// Test step spinner
	ui.StartStepSpinner("Building Docker image...")
	time.Sleep(2 * time.Second)
	ui.CompleteCurrentStep("Docker image built", true)

	ui.StartStepSpinner("Pushing to registry...")
	time.Sleep(2 * time.Second)
	ui.CompleteCurrentStep("Push failed", false)

	// Test step transition
	ui.StartStepSpinner("Starting migration...")
	time.Sleep(2 * time.Second)
	ui.CompleteStepAndStartNext("Migration complete", "Restarting services...")
	time.Sleep(2 * time.Second)
	ui.CompleteCurrentStep("Services restarted", true)
}
