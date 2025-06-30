# Hydra Workflow Engine

A powerful, distributed workflow orchestration engine for Go applications. Hydra enables you to build reliable, long-running business processes with automatic checkpointing, fault tolerance, and horizontal scaling.

## Features

✅ **Type-safe workflow steps** with Go generics  
✅ **Automatic checkpointing** - resume workflows after failures  
✅ **Distributed execution** with worker coordination  
✅ **Durable sleep operations** that don't block workers  
✅ **Cron scheduling** for time-based workflows  
✅ **Database agnostic** with pluggable storage backends  
✅ **Comprehensive observability** with Prometheus metrics  
✅ **Namespace isolation** for multi-tenancy  
✅ **Unique input/output capture** for complete audit trails  

## What Makes Hydra Different

Unlike other workflow engines, Hydra captures **both step inputs AND outputs** for complete visibility and debugging. This enables:

- **Complete audit trails** - see exactly what data was processed
- **Enhanced debugging** - inspect step inputs/outputs for failed workflows  
- **Replay capabilities** - rerun workflows with identical data
- **Data lineage tracking** - trace data flow through complex processes

## Quick Start

### Installation

```bash
go get github.com/unkeyed/unkey/go/pkg/hydra
```

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/unkeyed/unkey/go/pkg/hydra"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

// Define your payload types
type UserRegistrationPayload struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    Name   string `json:"name"`
}

func (u UserRegistrationPayload) Marshal() ([]byte, error) {
    return json.Marshal(u)
}

func (u *UserRegistrationPayload) Unmarshal(data []byte) error {
    return json.Unmarshal(data, u)
}

type EmailRequest struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

func (e EmailRequest) Marshal() ([]byte, error) {
    return json.Marshal(e)
}

func (e *EmailRequest) Unmarshal(data []byte) error {
    return json.Unmarshal(data, e)
}

type EmailResponse struct {
    MessageID string `json:"message_id"`
    Status    string `json:"status"`
}

func (e EmailResponse) Marshal() ([]byte, error) {
    return json.Marshal(e)
}

func (e *EmailResponse) Unmarshal(data []byte) error {
    return json.Unmarshal(data, e)
}

// Define your workflow
func UserOnboardingWorkflow(ctx hydra.WorkflowContext, payload hydra.Payload) error {
    user := payload.(*UserRegistrationPayload)
    
    // Step 1: Send welcome email
    emailResp, err := hydra.Step(ctx, "send-welcome-email", &EmailRequest{
        To:      user.Email,
        Subject: "Welcome to our platform!",
        Body:    fmt.Sprintf("Hello %s, welcome aboard!", user.Name),
    }, func(ctx context.Context, req *EmailRequest) (*EmailResponse, error) {
        // Your email sending logic here
        return &EmailResponse{
            MessageID: "msg-123",
            Status:    "sent",
        }, nil
    })
    if err != nil {
        return fmt.Errorf("failed to send welcome email: %w", err)
    }
    
    // Step 2: Wait 24 hours for user activation
    err = hydra.Sleep(ctx, 24*time.Hour)
    if err != nil {
        return fmt.Errorf("sleep failed: %w", err)
    }
    
    // Step 3: Send follow-up email
    _, err = hydra.Step(ctx, "send-followup-email", &EmailRequest{
        To:      user.Email,
        Subject: "Getting started guide",
        Body:    "Here's how to get the most out of our platform...",
    }, func(ctx context.Context, req *EmailRequest) (*EmailResponse, error) {
        // Follow-up email logic
        return &EmailResponse{
            MessageID: "msg-124", 
            Status:    "sent",
        }, nil
    })
    
    return err
}

func main() {
    // Setup database
    db, err := gorm.Open(postgres.Open("your-database-url"), &gorm.Config{})
    if err != nil {
        panic(err)
    }
    
    // Create Hydra store and instance
    store := hydra.NewGORMStore(db)
    config := &hydra.Config{
        Store:     store,
        Namespace: "production",
    }
    
    h := hydra.New(config)
    
    // Register workflow
    err = h.RegisterWorkflow("user-onboarding", UserOnboardingWorkflow)
    if err != nil {
        panic(err)
    }
    
    // Start worker in background
    ctx := context.Background()
    worker, err := h.StartWorker(ctx, hydra.WorkerConfig{
        WorkerID:     "worker-1",
        Concurrency:  20,
        PollInterval: 5 * time.Second,
    })
    if err != nil {
        panic(err)
    }
    defer worker.Shutdown(ctx)
    
    // Start workflow
    payload := &UserRegistrationPayload{
        UserID: "user-123",
        Email:  "john@example.com", 
        Name:   "John Doe",
    }
    
    workflowID, err := h.StartWorkflow(ctx, "user-onboarding", payload)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Started workflow: %s\n", workflowID)
}
```

## Core Concepts

### Workflows
Workflows are deterministic functions that orchestrate business processes. They're composed of Steps and can Sleep between operations.

### Steps
Steps are the atomic units of work within workflows. They're automatically checkpointed, meaning if a workflow fails and retries, completed steps are skipped.

### Workers
Workers are the execution engines that process workflows. Multiple workers can run concurrently for horizontal scaling.

### Payloads
All data in Hydra must implement the `Payload` interface for serialization. This ensures type safety and proper persistence.

### Sleep Operations
Unlike blocking sleeps, Hydra's `Sleep` function durably suspends workflows without tying up worker resources.

## Advanced Usage

### Error Handling

```go
func RobustWorkflow(ctx hydra.WorkflowContext, payload hydra.Payload) error {
    result, err := hydra.Step(ctx, "api-call", request, func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
        resp, err := apiClient.Call(req)
        if err != nil {
            // Wrap errors for better debugging
            return nil, fmt.Errorf("API call failed: %w", err)
        }
        
        if resp.StatusCode >= 400 {
            return nil, fmt.Errorf("API returned error: %d", resp.StatusCode)
        }
        
        return resp, nil
    })
    
    if err != nil {
        return fmt.Errorf("workflow failed at API step: %w", err)
    }
    
    return nil
}
```

### Conditional Logic

```go
func ConditionalWorkflow(ctx hydra.WorkflowContext, payload hydra.Payload) error {
    // Decision step
    decision, err := hydra.Step(ctx, "evaluate-condition", input, decisionFunction)
    if err != nil {
        return err
    }
    
    // Conditional execution
    if decision.ShouldProcessA {
        _, err = hydra.Step(ctx, "process-a", decision, processA)
    } else {
        _, err = hydra.Step(ctx, "process-b", decision, processB)
    }
    
    return err
}
```

### Cron Scheduling

```go
// Schedule a workflow to run daily at 9 AM
err := h.CreateCronJob(
    "0 9 * * *",           // cron expression
    "daily-report",        // job name
    "GenerateReport",      // workflow name
)
```

## Configuration

### Hydra Configuration

```go
config := &hydra.Config{
    Store:     store,
    Namespace: "production",
}

h := hydra.New(config)

// Start workers with custom settings
worker, err := h.StartWorker(ctx, hydra.WorkerConfig{
    WorkerID:          "worker-1",
    Concurrency:       50,              // Max concurrent workflows
    PollInterval:      5 * time.Second, // How often to check for work
    HeartbeatInterval: 30 * time.Second,
    ClaimTimeout:      5 * time.Minute,
})
```

### Workflow Options

```go
// Start workflow with custom settings
workflowID, err := h.StartWorkflow(
    ctx,
    "my-workflow", 
    payload,
    hydra.WithMaxAttempts(5),
    hydra.WithTimeout(2*time.Hour),
    hydra.WithRetryBackoff(1*time.Minute),
)
```

## Production Deployment

### Database Setup

Hydra supports PostgreSQL and MySQL. Run database migrations:

```go
// Auto-migrate schema
store := hydra.NewGORMStore(db)
err := db.AutoMigrate(
    &hydra.WorkflowExecution{},
    &hydra.WorkflowStep{}, 
    &hydra.CronJob{},
    &hydra.Lease{},
)
```

### Multi-Worker Deployment

```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: workflows
      POSTGRES_USER: hydra
      POSTGRES_PASSWORD: secure-password
    ports:
      - "5432:5432"

  hydra-worker-1:
    build: .
    environment:
      DATABASE_URL: "postgres://hydra:secure-password@postgres:5432/workflows"
      WORKER_ID: "worker-1"
      NAMESPACE: "production"
      MAX_CONCURRENCY: "20"
    depends_on:
      - postgres

  hydra-worker-2:
    build: .
    environment:
      DATABASE_URL: "postgres://hydra:secure-password@postgres:5432/workflows"
      WORKER_ID: "worker-2" 
      NAMESPACE: "production"
      MAX_CONCURRENCY: "20"
    depends_on:
      - postgres
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hydra-workers
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hydra-worker
  template:
    metadata:
      labels:
        app: hydra-worker
    spec:
      containers:
      - name: hydra-worker
        image: your-registry/hydra-worker:latest
        env:
        - name: WORKER_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: hydra-secrets
              key: database-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Monitoring

### Prometheus Metrics

Hydra exposes comprehensive metrics:

- `hydra_workflows_total` - Total workflows processed
- `hydra_workflow_duration_seconds` - Workflow completion time
- `hydra_active_workflows` - Currently running workflows
- `hydra_step_duration_seconds` - Step execution time
- `hydra_database_operations_total` - Database operation counts

### Health Checks

```go
// Built-in health check endpoints
http.HandleFunc("/health", healthCheck)     // Liveness probe
http.HandleFunc("/ready", readinessCheck)   // Readiness probe  
http.HandleFunc("/metrics", promhttp.Handler()) // Prometheus metrics
```

## Best Practices

### Step Design
- Keep steps idempotent - they may be retried
- Use descriptive step names for debugging
- Handle errors gracefully with proper context

### Payload Design
- Use versioned payloads for schema evolution
- Keep payloads small - they're stored in the database
- Consider encryption for sensitive data

### Performance
- Monitor queue depth and worker utilization
- Scale workers horizontally for increased throughput
- Optimize database indexes for your access patterns

### Error Handling
- Distinguish between retryable and non-retryable errors
- Use structured logging for workflow events
- Set appropriate retry policies per workflow type

## API Reference

See the [complete API documentation](https://docs.unkey.com/architecture/hydra/api-reference) for detailed function signatures and examples.

## Examples

Explore real-world examples:

- [E-commerce Order Processing](https://docs.unkey.com/architecture/hydra/examples#e-commerce-order-processing)
- [User Onboarding Flow](https://docs.unkey.com/architecture/hydra/examples#user-onboarding-workflow)
- [Data Processing Pipeline](https://docs.unkey.com/architecture/hydra/examples#data-processing-pipeline)
- [Billing and Subscription Management](https://docs.unkey.com/architecture/hydra/examples#billing-workflow)

## Contributing

Hydra is part of the [Unkey](https://github.com/unkeyed/unkey) project. Contributions are welcome!

## License

Apache 2.0 - see [LICENSE](https://github.com/unkeyed/unkey/blob/main/LICENSE) for details.

## Support

- 📖 [Documentation](https://docs.unkey.com/architecture/hydra)
- 💬 [Discord Community](https://discord.gg/unkey)
- 🐛 [Issue Tracker](https://github.com/unkeyed/unkey/issues)
- 📧 [Email Support](mailto:support@unkey.com)