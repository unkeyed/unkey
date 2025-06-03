package deployment

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	DeploymentWorkflowName = "deployment-workflow"
	TaskQueue             = "deployment-task-queue"
)

type DeploymentWorkflowInput struct {
	Deployment *Deployment `json:"deployment"`
}

type DeploymentWorkflowResult struct {
	DeploymentID string           `json:"deployment_id"`
	Status       DeploymentStatus `json:"status"`
	Error        string           `json:"error,omitempty"`
}

// DeploymentWorkflow orchestrates the entire deployment process
func DeploymentWorkflow(ctx workflow.Context, input DeploymentWorkflowInput) (*DeploymentWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	deployment := input.Deployment

	logger.Info("Starting deployment workflow", "deployment_id", deployment.ID)

	// Set up activity options with retries
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:   5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2.0,
			MaximumInterval:        5 * time.Minute,
			MaximumAttempts:        5,
			NonRetryableErrorTypes: []string{"NonRetryableError"},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := &DeploymentWorkflowResult{
		DeploymentID: deployment.ID,
		Status:       StatusPending,
	}

	// Step 1: Download source code
	if err := executeStep(ctx, deployment, StepSourceDownload, "DownloadSourceActivity"); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("Source download failed: %v", err)
		return result, err
	}

	// Step 2: Build container image
	if err := executeStep(ctx, deployment, StepBuildImage, "BuildImageActivity"); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("Image build failed: %v", err)
		return result, err
	}

	// Step 3: Push image to registry
	if err := executeStep(ctx, deployment, StepPushImage, "PushImageActivity"); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("Image push failed: %v", err)
		return result, err
	}

	// Step 4: Provision infrastructure resources
	if err := executeStep(ctx, deployment, StepProvision, "ProvisionResourcesActivity"); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("Resource provisioning failed: %v", err)
		return result, err
	}

	// Step 5: Deploy application
	if err := executeStep(ctx, deployment, StepDeploy, "DeployApplicationActivity"); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("Application deployment failed: %v", err)
		return result, err
	}

	// Step 6: Health check with retries
	if err := executeHealthCheck(ctx, deployment); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("Health check failed: %v", err)
		// Attempt rollback
		if rollbackErr := executeStep(ctx, deployment, StepDeploy, "RollbackActivity"); rollbackErr != nil {
			logger.Error("Rollback failed", "error", rollbackErr)
		}
		return result, err
	}

	// Step 7: Route traffic
	if err := executeStep(ctx, deployment, StepTrafficRoute, "RouteTrafficActivity"); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("Traffic routing failed: %v", err)
		return result, err
	}

	// Mark deployment as successful
	if err := updateDeploymentStatus(ctx, deployment.ID, StatusRunning); err != nil {
		logger.Error("Failed to update deployment status", "error", err)
	}

	result.Status = StatusRunning
	logger.Info("Deployment workflow completed successfully", "deployment_id", deployment.ID)
	return result, nil
}

func executeStep(ctx workflow.Context, deployment *Deployment, step DeploymentStep, activityName string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing deployment step", "step", step, "deployment_id", deployment.ID)

	// Update current step
	if err := updateCurrentStep(ctx, deployment.ID, step); err != nil {
		logger.Error("Failed to update current step", "step", step, "error", err)
	}

	var result StepResult
	err := workflow.ExecuteActivity(ctx, activityName, deployment).Get(ctx, &result)
	if err != nil {
		logger.Error("Step execution failed", "step", step, "error", err)
		// Record step failure
		recordStepResult(ctx, deployment.ID, StepResult{
			Step:        step,
			Status:      StepStatusFailed,
			StartedAt:   workflow.Now(ctx),
			CompletedAt: &[]time.Time{workflow.Now(ctx)}[0],
			Error: &StepError{
				Code:       "EXECUTION_FAILED",
				Message:    err.Error(),
				Retryable:  true,
				OccurredAt: workflow.Now(ctx),
			},
		})
		return err
	}

	// Record successful step
	recordStepResult(ctx, deployment.ID, result)
	return nil
}

func executeHealthCheck(ctx workflow.Context, deployment *Deployment) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting health check", "deployment_id", deployment.ID)

	// Custom retry policy for health checks
	healthCheckAO := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:   30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 1.5,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    10, // More retries for health checks
		},
	}
	healthCtx := workflow.WithActivityOptions(ctx, healthCheckAO)

	return executeStep(healthCtx, deployment, StepHealthCheck, "HealthCheckActivity")
}

func updateCurrentStep(ctx workflow.Context, deploymentID string, step DeploymentStep) error {
	var result interface{}
	return workflow.ExecuteActivity(ctx, "UpdateCurrentStepActivity", deploymentID, step).Get(ctx, &result)
}

func updateDeploymentStatus(ctx workflow.Context, deploymentID string, status DeploymentStatus) error {
	var result interface{}
	return workflow.ExecuteActivity(ctx, "UpdateDeploymentStatusActivity", deploymentID, status).Get(ctx, &result)
}

func recordStepResult(ctx workflow.Context, deploymentID string, stepResult StepResult) error {
	var result interface{}
	return workflow.ExecuteActivity(ctx, "RecordStepResultActivity", deploymentID, stepResult).Get(ctx, &result)
}

// Activity implementations
type DeploymentActivities struct {
	orchestrator DeploymentOrchestrator
	publisher    EventPublisher
	logStreamer  LogStreamer
	resourceMgr  ResourceManager
}

func NewDeploymentActivities(
	orchestrator DeploymentOrchestrator,
	publisher EventPublisher,
	logStreamer LogStreamer,
	resourceMgr ResourceManager,
) *DeploymentActivities {
	return &DeploymentActivities{
		orchestrator: orchestrator,
		publisher:    publisher,
		logStreamer:  logStreamer,
		resourceMgr:  resourceMgr,
	}
}

func (a *DeploymentActivities) DownloadSourceActivity(ctx context.Context, deployment *Deployment) (StepResult, error) {
	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	logger.Info("Downloading source code", "deployment_id", deployment.ID, "source_type", deployment.Source.Type)

	// Simulate source download logic
	// In real implementation, this would download from Git, S3, etc.
	time.Sleep(2 * time.Second) // Simulate work

	return StepResult{
		Step:        StepSourceDownload,
		Status:      StepStatusCompleted,
		StartedAt:   startTime,
		CompletedAt: &[]time.Time{time.Now()}[0],
		Duration:    &Duration{time.Since(startTime)},
		Attempts:    1,
		Output:      "Source code downloaded successfully",
	}, nil
}

func (a *DeploymentActivities) BuildImageActivity(ctx context.Context, deployment *Deployment) (StepResult, error) {
	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	logger.Info("Building container image", "deployment_id", deployment.ID)

	// Heartbeat for long-running operation
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				activity.RecordHeartbeat(ctx, "Building image...")
			}
		}
	}()

	// Simulate image build (in reality, this would call Docker API or buildkit)
	time.Sleep(5 * time.Second)

	deployment.Metadata.ImageURI = fmt.Sprintf("registry.example.com/%s/%s:latest", deployment.CustomerID, deployment.ProjectID)

	return StepResult{
		Step:        StepBuildImage,
		Status:      StepStatusCompleted,
		StartedAt:   startTime,
		CompletedAt: &[]time.Time{time.Now()}[0],
		Duration:    &Duration{time.Since(startTime)},
		Attempts:    1,
		Output:      fmt.Sprintf("Image built: %s", deployment.Metadata.ImageURI),
	}, nil
}

func (a *DeploymentActivities) PushImageActivity(ctx context.Context, deployment *Deployment) (StepResult, error) {
	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	logger.Info("Pushing container image", "deployment_id", deployment.ID, "image", deployment.Metadata.ImageURI)

	// Simulate image push
	time.Sleep(3 * time.Second)

	return StepResult{
		Step:        StepPushImage,
		Status:      StepStatusCompleted,
		StartedAt:   startTime,
		CompletedAt: &[]time.Time{time.Now()}[0],
		Duration:    &Duration{time.Since(startTime)},
		Attempts:    1,
		Output:      "Image pushed to registry",
	}, nil
}

func (a *DeploymentActivities) ProvisionResourcesActivity(ctx context.Context, deployment *Deployment) (StepResult, error) {
	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	logger.Info("Provisioning resources", "deployment_id", deployment.ID)

	if err := a.resourceMgr.ProvisionResources(ctx, deployment); err != nil {
		return StepResult{}, fmt.Errorf("resource provisioning failed: %w", err)
	}

	return StepResult{
		Step:        StepProvision,
		Status:      StepStatusCompleted,
		StartedAt:   startTime,
		CompletedAt: &[]time.Time{time.Now()}[0],
		Duration:    &Duration{time.Since(startTime)},
		Attempts:    1,
		Output:      "Resources provisioned successfully",
	}, nil
}

func (a *DeploymentActivities) DeployApplicationActivity(ctx context.Context, deployment *Deployment) (StepResult, error) {
	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	logger.Info("Deploying application", "deployment_id", deployment.ID)

	// Simulate application deployment (Kubernetes, Docker Swarm, etc.)
	time.Sleep(4 * time.Second)

	deployment.Metadata.ServiceName = fmt.Sprintf("svc-%s-%s", deployment.CustomerID, deployment.ProjectID)
	deployment.Metadata.InternalURL = fmt.Sprintf("http://%s.internal:8080", deployment.Metadata.ServiceName)
	deployment.Metadata.ExternalURL = fmt.Sprintf("https://%s.example.com", deployment.ProjectID)

	return StepResult{
		Step:        StepDeploy,
		Status:      StepStatusCompleted,
		StartedAt:   startTime,
		CompletedAt: &[]time.Time{time.Now()}[0],
		Duration:    &Duration{time.Since(startTime)},
		Attempts:    1,
		Output:      fmt.Sprintf("Application deployed at %s", deployment.Metadata.ExternalURL),
	}, nil
}

func (a *DeploymentActivities) HealthCheckActivity(ctx context.Context, deployment *Deployment) (StepResult, error) {
	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	logger.Info("Performing health check", "deployment_id", deployment.ID)

	// Simulate health check
	for i := 0; i < 5; i++ {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Health check attempt %d/5", i+1))
		time.Sleep(2 * time.Second)
		
		// Simulate random health check success/failure
		if i >= 2 { // Succeed after 3 attempts
			break
		}
	}

	return StepResult{
		Step:        StepHealthCheck,
		Status:      StepStatusCompleted,
		StartedAt:   startTime,
		CompletedAt: &[]time.Time{time.Now()}[0],
		Duration:    &Duration{time.Since(startTime)},
		Attempts:    1,
		Output:      "Health check passed",
	}, nil
}

func (a *DeploymentActivities) RouteTrafficActivity(ctx context.Context, deployment *Deployment) (StepResult, error) {
	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	logger.Info("Routing traffic", "deployment_id", deployment.ID)

	// Simulate traffic routing (load balancer configuration, DNS updates, etc.)
	time.Sleep(2 * time.Second)

	return StepResult{
		Step:        StepTrafficRoute,
		Status:      StepStatusCompleted,
		StartedAt:   startTime,
		CompletedAt: &[]time.Time{time.Now()}[0],
		Duration:    &Duration{time.Since(startTime)},
		Attempts:    1,
		Output:      "Traffic routed successfully",
	}, nil
}

func (a *DeploymentActivities) UpdateCurrentStepActivity(ctx context.Context, deploymentID string, step DeploymentStep) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating current step", "deployment_id", deploymentID, "step", step)
	// Implementation would update database
	return nil
}

func (a *DeploymentActivities) UpdateDeploymentStatusActivity(ctx context.Context, deploymentID string, status DeploymentStatus) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating deployment status", "deployment_id", deploymentID, "status", status)
	// Implementation would update database
	return nil
}

func (a *DeploymentActivities) RecordStepResultActivity(ctx context.Context, deploymentID string, stepResult StepResult) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Recording step result", "deployment_id", deploymentID, "step", stepResult.Step, "status", stepResult.Status)
	// Implementation would update database and publish events
	return nil
}

func (a *DeploymentActivities) RollbackActivity(ctx context.Context, deployment *Deployment) (StepResult, error) {
	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	logger.Info("Rolling back deployment", "deployment_id", deployment.ID)

	// Simulate rollback logic
	time.Sleep(3 * time.Second)

	return StepResult{
		Step:        StepDeploy,
		Status:      StepStatusCompleted,
		StartedAt:   startTime,
		CompletedAt: &[]time.Time{time.Now()}[0],
		Duration:    &Duration{time.Since(startTime)},
		Attempts:    1,
		Output:      "Deployment rolled back successfully",
	}, nil
}