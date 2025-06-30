package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
)

type SendEmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type EmailWorkflow struct{}

func (w *EmailWorkflow) Name() string {
	return "send-email"
}

func (w *EmailWorkflow) Run(ctx hydra.WorkflowContext, req SendEmailRequest) error {
	result, err := hydra.Step(ctx, "send-email", func(stepCtx context.Context) (string, error) {
		fmt.Printf("Sending email to %s: %s\n", req.To, req.Subject)
		return "email-sent-successfully", nil
	})

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("Sleeping at %s\n", time.Now())
	err = hydra.Sleep(ctx, time.Second)
	if err != nil {
		return fmt.Errorf("failed to sleep: %w", err)
	}
	fmt.Printf("Email step completed: %s - %s\n", result, time.Now())
	return nil
}

func (w *EmailWorkflow) Start(h *hydra.Hydra, ctx context.Context, req SendEmailRequest) (string, error) {
	return h.StartWorkflow(ctx, w.Name(), req)
}

func main() {
	// Create hydra instance with in-memory SQLite store
	store, err := gorm.NewSQLiteStore("", nil)
	if err != nil {
		log.Fatal("Failed to create store:", err)
	}

	h := hydra.New(hydra.Config{
		Store: store,
	})

	wf := &EmailWorkflow{}

	// Start worker with workflows registered
	worker, err := h.StartWorker(context.Background(), hydra.WorkerConfig{
		Concurrency: 1,
		Workflows:   []hydra.WorkflowRunner{wf},
	})
	if err != nil {
		log.Fatal("Failed to start worker:", err)
	}
	defer worker.Shutdown(context.Background())

	// Give worker a moment to start
	time.Sleep(100 * time.Millisecond)

	// Start a workflow execution
	request := SendEmailRequest{
		To:      "user@example.com",
		Subject: "Welcome!",
		Body:    "Welcome to our service!",
	}

	executionID, err := wf.Start(h, context.Background(), request)
	if err != nil {
		log.Fatal("Failed to start workflow:", err)
	}

	fmt.Printf("Started workflow execution: %s\n", executionID)

	// Give time for workflow to complete
	time.Sleep(10 * time.Second)
	fmt.Println("Done!")
}
