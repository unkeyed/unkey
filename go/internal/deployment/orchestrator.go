package deployment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
)

type DatabaseDeploymentOrchestrator struct {
	db               *sql.DB
	temporalClient   client.Client
	eventPublisher   EventPublisher
	logStreamer      LogStreamer
	resourceManager  ResourceManager
}

func NewDatabaseDeploymentOrchestrator(
	db *sql.DB,
	temporalClient client.Client,
	eventPublisher EventPublisher,
	logStreamer LogStreamer,
	resourceManager ResourceManager,
) *DatabaseDeploymentOrchestrator {
	return &DatabaseDeploymentOrchestrator{
		db:               db,
		temporalClient:   temporalClient,
		eventPublisher:   eventPublisher,
		logStreamer:      logStreamer,
		resourceManager:  resourceManager,
	}
}

func (o *DatabaseDeploymentOrchestrator) Deploy(ctx context.Context, deployment *Deployment) error {
	// Store deployment in database
	if err := o.storeDeployment(ctx, deployment); err != nil {
		return fmt.Errorf("failed to store deployment: %w", err)
	}

	// Publish deployment started event
	if err := o.publishEvent(ctx, &DeploymentEvent{
		Type:         "deployment.started",
		DeploymentID: deployment.ID,
		CustomerID:   deployment.CustomerID,
		ProjectID:    deployment.ProjectID,
		Status:       StatusPending,
		Timestamp:    time.Now(),
	}); err != nil {
		// Log error but don't fail deployment
		fmt.Printf("Failed to publish deployment started event: %v\n", err)
	}

	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("deployment-%s", deployment.ID),
		TaskQueue: TaskQueue,
		WorkflowExecutionTimeout: 2 * time.Hour,
		WorkflowIDReusePolicy: client.WorkflowIDReusePolicyAllowDuplicate,
	}

	input := DeploymentWorkflowInput{
		Deployment: deployment,
	}

	_, err := o.temporalClient.ExecuteWorkflow(ctx, workflowOptions, DeploymentWorkflowName, input)
	if err != nil {
		// Update deployment status to failed
		deployment.Status = StatusFailed
		o.updateDeploymentStatus(ctx, deployment.ID, StatusFailed)
		
		o.publishEvent(ctx, &DeploymentEvent{
			Type:         "deployment.failed",
			DeploymentID: deployment.ID,
			CustomerID:   deployment.CustomerID,
			ProjectID:    deployment.ProjectID,
			Status:       StatusFailed,
			Data:         map[string]interface{}{"error": err.Error()},
			Timestamp:    time.Now(),
		})
		
		return fmt.Errorf("failed to start deployment workflow: %w", err)
	}

	return nil
}

func (o *DatabaseDeploymentOrchestrator) Cancel(ctx context.Context, deploymentID string) error {
	// Get deployment from database
	deployment, err := o.GetStatus(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Cancel Temporal workflow
	workflowID := fmt.Sprintf("deployment-%s", deploymentID)
	err = o.temporalClient.CancelWorkflow(ctx, workflowID, "")
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	// Update deployment status
	if err := o.updateDeploymentStatus(ctx, deploymentID, StatusCanceled); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	// Publish cancellation event
	o.publishEvent(ctx, &DeploymentEvent{
		Type:         "deployment.canceled",
		DeploymentID: deployment.ID,
		CustomerID:   deployment.CustomerID,
		ProjectID:    deployment.ProjectID,
		Status:       StatusCanceled,
		Timestamp:    time.Now(),
	})

	return nil
}

func (o *DatabaseDeploymentOrchestrator) GetStatus(ctx context.Context, deploymentID string) (*Deployment, error) {
	query := `
		SELECT id, customer_id, project_id, status, current_step, source_config, 
		       build_config, runtime_config, resource_config, metadata, steps,
		       created_at, updated_at, completed_at
		FROM deployments 
		WHERE id = ?
	`

	var deployment Deployment
	var sourceConfigJSON, buildConfigJSON, runtimeConfigJSON, resourceConfigJSON, metadataJSON, stepsJSON []byte
	var completedAt sql.NullTime

	err := o.db.QueryRowContext(ctx, query, deploymentID).Scan(
		&deployment.ID,
		&deployment.CustomerID,
		&deployment.ProjectID,
		&deployment.Status,
		&deployment.CurrentStep,
		&sourceConfigJSON,
		&buildConfigJSON,
		&runtimeConfigJSON,
		&resourceConfigJSON,
		&metadataJSON,
		&stepsJSON,
		&deployment.CreatedAt,
		&deployment.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("deployment not found: %s", deploymentID)
		}
		return nil, fmt.Errorf("failed to query deployment: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(sourceConfigJSON, &deployment.Source); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source config: %w", err)
	}
	if err := json.Unmarshal(buildConfigJSON, &deployment.Build); err != nil {
		return nil, fmt.Errorf("failed to unmarshal build config: %w", err)
	}
	if err := json.Unmarshal(runtimeConfigJSON, &deployment.Runtime); err != nil {
		return nil, fmt.Errorf("failed to unmarshal runtime config: %w", err)
	}
	if err := json.Unmarshal(resourceConfigJSON, &deployment.Resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource config: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &deployment.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	if err := json.Unmarshal(stepsJSON, &deployment.Steps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal steps: %w", err)
	}

	if completedAt.Valid {
		deployment.CompletedAt = &completedAt.Time
	}

	return &deployment, nil
}

func (o *DatabaseDeploymentOrchestrator) ListDeployments(ctx context.Context, customerID string, opts ListOptions) ([]*Deployment, error) {
	query := `
		SELECT id, customer_id, project_id, status, current_step, source_config, 
		       build_config, runtime_config, resource_config, metadata, steps,
		       created_at, updated_at, completed_at
		FROM deployments 
		WHERE customer_id = ?
	`
	args := []interface{}{customerID}

	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := o.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployments: %w", err)
	}
	defer rows.Close()

	var deployments []*Deployment
	for rows.Next() {
		var deployment Deployment
		var sourceConfigJSON, buildConfigJSON, runtimeConfigJSON, resourceConfigJSON, metadataJSON, stepsJSON []byte
		var completedAt sql.NullTime

		err := rows.Scan(
			&deployment.ID,
			&deployment.CustomerID,
			&deployment.ProjectID,
			&deployment.Status,
			&deployment.CurrentStep,
			&sourceConfigJSON,
			&buildConfigJSON,
			&runtimeConfigJSON,
			&resourceConfigJSON,
			&metadataJSON,
			&stepsJSON,
			&deployment.CreatedAt,
			&deployment.UpdatedAt,
			&completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}

		// Unmarshal JSON fields
		json.Unmarshal(sourceConfigJSON, &deployment.Source)
		json.Unmarshal(buildConfigJSON, &deployment.Build)
		json.Unmarshal(runtimeConfigJSON, &deployment.Runtime)
		json.Unmarshal(resourceConfigJSON, &deployment.Resources)
		json.Unmarshal(metadataJSON, &deployment.Metadata)
		json.Unmarshal(stepsJSON, &deployment.Steps)

		if completedAt.Valid {
			deployment.CompletedAt = &completedAt.Time
		}

		deployments = append(deployments, &deployment)
	}

	return deployments, nil
}

func (o *DatabaseDeploymentOrchestrator) Rollback(ctx context.Context, deploymentID string, targetDeploymentID string) error {
	// Get current deployment
	currentDeployment, err := o.GetStatus(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get current deployment: %w", err)
	}

	// Get target deployment to rollback to
	targetDeployment, err := o.GetStatus(ctx, targetDeploymentID)
	if err != nil {
		return fmt.Errorf("failed to get target deployment: %w", err)
	}

	// Validate rollback is possible
	if currentDeployment.CustomerID != targetDeployment.CustomerID ||
		currentDeployment.ProjectID != targetDeployment.ProjectID {
		return fmt.Errorf("cannot rollback between different customers or projects")
	}

	if targetDeployment.Status != StatusRunning {
		return fmt.Errorf("target deployment is not in running state")
	}

	// Create rollback deployment
	rollbackDeployment := &Deployment{
		ID:          fmt.Sprintf("rollback-%s-%d", deploymentID, time.Now().Unix()),
		CustomerID:  currentDeployment.CustomerID,
		ProjectID:   currentDeployment.ProjectID,
		Status:      StatusPending,
		CurrentStep: StepSourceDownload,
		Source:      targetDeployment.Source,
		Build:       targetDeployment.Build,
		Runtime:     targetDeployment.Runtime,
		Resources:   targetDeployment.Resources,
		Metadata:    DeploymentMeta{},
		Steps:       []StepResult{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Deploy the rollback
	if err := o.Deploy(ctx, rollbackDeployment); err != nil {
		return fmt.Errorf("failed to deploy rollback: %w", err)
	}

	// Update original deployment status
	if err := o.updateDeploymentStatus(ctx, deploymentID, StatusRolledBack); err != nil {
		return fmt.Errorf("failed to update original deployment status: %w", err)
	}

	// Publish rollback event
	o.publishEvent(ctx, &DeploymentEvent{
		Type:         "deployment.rolled_back",
		DeploymentID: deploymentID,
		CustomerID:   currentDeployment.CustomerID,
		ProjectID:    currentDeployment.ProjectID,
		Status:       StatusRolledBack,
		Data: map[string]interface{}{
			"target_deployment_id": targetDeploymentID,
			"rollback_deployment_id": rollbackDeployment.ID,
		},
		Timestamp:    time.Now(),
	})

	return nil
}

func (o *DatabaseDeploymentOrchestrator) storeDeployment(ctx context.Context, deployment *Deployment) error {
	sourceConfigJSON, _ := json.Marshal(deployment.Source)
	buildConfigJSON, _ := json.Marshal(deployment.Build)
	runtimeConfigJSON, _ := json.Marshal(deployment.Runtime)
	resourceConfigJSON, _ := json.Marshal(deployment.Resources)
	metadataJSON, _ := json.Marshal(deployment.Metadata)
	stepsJSON, _ := json.Marshal(deployment.Steps)

	query := `
		INSERT INTO deployments (
			id, customer_id, project_id, status, current_step, source_config,
			build_config, runtime_config, resource_config, metadata, steps,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := o.db.ExecContext(ctx, query,
		deployment.ID,
		deployment.CustomerID,
		deployment.ProjectID,
		deployment.Status,
		deployment.CurrentStep,
		sourceConfigJSON,
		buildConfigJSON,
		runtimeConfigJSON,
		resourceConfigJSON,
		metadataJSON,
		stepsJSON,
		deployment.CreatedAt,
		deployment.UpdatedAt,
	)

	return err
}

func (o *DatabaseDeploymentOrchestrator) updateDeploymentStatus(ctx context.Context, deploymentID string, status DeploymentStatus) error {
	query := `UPDATE deployments SET status = ?, updated_at = ? WHERE id = ?`
	_, err := o.db.ExecContext(ctx, query, status, time.Now(), deploymentID)
	return err
}

func (o *DatabaseDeploymentOrchestrator) publishEvent(ctx context.Context, event *DeploymentEvent) error {
	if o.eventPublisher == nil {
		return nil // No publisher configured
	}
	return o.eventPublisher.Publish(ctx, event)
}