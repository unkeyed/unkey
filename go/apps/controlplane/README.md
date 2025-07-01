# Controlplane Service

The controlplane service is a background workflow processing service built on top of the Hydra workflow engine. It provides reliable, scalable workflow execution with exactly-once guarantees, automatic retries, and crash recovery.

## Overview

The controlplane service runs workflows in the background without exposing any public API endpoints. It's designed for:

- Background data processing
- Async operations that need reliability guarantees
- Long-running business processes
- Scheduled or triggered workflows
- Operations that require exactly-once execution

## Getting Started

### 1. Running the Service

```bash
# Basic setup with required database connection
go run main.go controlplane --database-primary "user:pass@localhost:3306/unkey?parseTime=true"

# Production setup with all options
go run main.go controlplane \
  --database-primary "user:pass@db-primary:3306/unkey?parseTime=true" \
  --database-replica "user:pass@db-replica:3306/unkey?parseTime=true" \
  --worker-concurrency 10 \
  --worker-poll-interval "500ms" \
  --worker-heartbeat-interval "30s" \
  --worker-claim-timeout "5m" \
  --otel \
  --prometheus-port 9090 \
  --region "us-east-1" \
  --instance-id "controlplane-001"
```

### 2. Environment Variables

All CLI flags can be set via environment variables:

```bash
export UNKEY_DATABASE_PRIMARY="user:pass@localhost:3306/unkey?parseTime=true"
export UNKEY_WORKER_CONCURRENCY=10
export UNKEY_WORKER_POLL_INTERVAL="500ms"
export UNKEY_OTEL=true
export UNKEY_PROMETHEUS_PORT=9090
```

### 3. Creating Your First Workflow

1. Create a new workflow file in `workflows/`:

```go
package workflows

import (
    "context"
    "github.com/unkeyed/unkey/go/pkg/hydra"
    "github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type MyWorkflow struct {
    logger logging.Logger
}

func NewMyWorkflow(logger logging.Logger) *MyWorkflow {
    return &MyWorkflow{logger: logger}
}

func (w *MyWorkflow) Name() string {
    return "my-workflow"
}

type MyRequest struct {
    UserID string `json:"user_id"`
    Action string `json:"action"`
}

func (w *MyWorkflow) Run(ctx hydra.WorkflowContext, req MyRequest) error {
    // Step 1: Validate
    _, err := hydra.Step(ctx, "validate", func(stepCtx context.Context) (string, error) {
        if req.UserID == "" {
            return "", fmt.Errorf("user_id required")
        }
        return "valid", nil
    })
    if err != nil {
        return err
    }

    // Step 2: Process
    result, err := hydra.Step(ctx, "process", func(stepCtx context.Context) (string, error) {
        // Your business logic here
        w.logger.Info("processing", "user_id", req.UserID, "action", req.Action)
        return "processed", nil
    })
    if err != nil {
        return err
    }

    w.logger.Info("workflow completed", "result", result)
    return nil
}
```

2. Register your workflow in `run.go`:

```go
func registerWorkflows(worker hydra.Worker, logger logging.Logger) error {
    myWorkflow := workflows.NewMyWorkflow(logger)
    err := hydra.RegisterWorkflow(worker, myWorkflow)
    if err != nil {
        return fmt.Errorf("unable to register my workflow: %w", err)
    }

    logger.Info("workflows registered successfully")
    return nil
}
```

3. Start workflows from other services:

```go
// From your API service or other applications
req := workflows.MyRequest{
    UserID: "user123",
    Action: "process_payment",
}

workflowID, err := engine.StartWorkflow(ctx, "my-workflow", req)
if err != nil {
    return fmt.Errorf("failed to start workflow: %w", err)
}
```

## Configuration

### Worker Configuration

- **Concurrency**: Number of workflows processed simultaneously
- **Poll Interval**: How often to check for new workflows (lower = more responsive, higher = less DB load)
- **Heartbeat Interval**: How often to update workflow leases (should be much less than claim timeout)
- **Claim Timeout**: How long a worker holds a workflow before others can take over

### Database Setup

The service requires access to the same database as your main application. Hydra will automatically create the necessary tables:

- `workflow_executions`: Stores workflow state and metadata
- `workflow_steps`: Stores individual step execution results
- `leases`: Manages workflow ownership between workers

### Observability

- **Prometheus**: Enable with `--prometheus-port` for metrics
- **OpenTelemetry**: Enable with `--otel` for distributed tracing
- **Structured Logging**: JSON logs with correlation IDs

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Service   │    │  Other Services │    │  Scheduled Jobs │
│                 │    │                 │    │                 │
│  Starts         │    │  Starts         │    │  Starts         │
│  Workflows      │    │  Workflows      │    │  Workflows      │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼──────────────┐
                    │        Database            │
                    │   (Workflow Storage)       │
                    └─────────────┬──────────────┘
                                  │
         ┌────────────────────────┼────────────────────────┐
         │                       │                        │
┌────────▼────────┐    ┌─────────▼────────┐    ┌─────────▼────────┐
│  Controlplane   │    │  Controlplane    │    │  Controlplane    │
│   Worker 1      │    │   Worker 2       │    │   Worker N       │
│                 │    │                  │    │                  │
│ Processes       │    │ Processes        │    │ Processes        │
│ Workflows       │    │ Workflows        │    │ Workflows        │
└─────────────────┘    └──────────────────┘    └──────────────────┘
```

## Best Practices

### Workflow Design

1. **Idempotent Steps**: Each step should be safe to retry
2. **Small Steps**: Break complex logic into smaller, manageable steps
3. **Error Handling**: Always handle and log errors appropriately
4. **Timeouts**: Use context timeouts for external calls

### Performance

1. **Right-size Concurrency**: Start with 5-10, monitor CPU/memory usage
2. **Optimize Poll Interval**: Balance responsiveness vs database load
3. **Database Indexing**: Ensure proper indexes on workflow tables
4. **Resource Limits**: Set appropriate memory/CPU limits in production

### Monitoring

1. **Workflow Metrics**: Track completion rates, error rates, duration
2. **Worker Health**: Monitor worker uptime and heartbeat status
3. **Database Performance**: Watch for slow queries and lock contention
4. **Queue Depth**: Monitor pending workflow counts

## Built-in Workflows

### Quota Check Workflow

The controlplane service includes a built-in quota check workflow that monitors workspace usage and sends notifications when quotas are exceeded. This replaces the standalone `quotacheck` command with a scheduled workflow.

**Configuration:**

```bash
# Required for quota check workflow
export UNKEY_BUSINESS_DATABASE_PRIMARY="user:pass@mysql:3306/unkey?parseTime=true"
export UNKEY_CLICKHOUSE_URL="clickhouse://user:pass@clickhouse:9000/unkey"

# Optional for notifications
export UNKEY_SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."

# Run the service
./unkey controlplane
```

**Features:**
- **Automatic Scheduling**: Runs monthly on the 1st at 9:00 AM UTC
- **Slack Notifications**: Sends detailed alerts for quota violations
- **Exactly-once Execution**: Guaranteed workflow execution with crash recovery
- **Step-by-step Processing**: Validates inputs, processes workspaces, sends summaries
- **Error Resilience**: Continues processing other workspaces if one fails

**Manual Execution:**

```go
// You can also trigger quota checks manually
workflowID, err := engine.StartWorkflow(ctx, "quota-check", workflows.QuotaCheckRequest{
    Year:  2024,
    Month: 6, // June
})
```

## Example Workflows

See `workflows/example.go` for a complete example workflow that demonstrates:

- Input validation
- Multi-step processing
- Error handling
- Logging and observability
- Exactly-once execution guarantees

## Deployment

The controlplane service is designed to run as a long-lived service in your infrastructure:

- **Kubernetes**: Deploy as a Deployment with multiple replicas
- **Docker**: Run as a container with restart policies
- **Systemd**: Run as a system service with auto-restart
- **Process Managers**: Use supervisord, pm2, or similar

Multiple instances can run simultaneously and will automatically coordinate workflow processing through the database.