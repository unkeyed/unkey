# Hydra 🌊

> **Distributed workflow orchestration engine for Go**

Hydra is a robust, scalable workflow orchestration engine designed for reliable execution of multi-step business processes. Built with exactly-once execution guarantees, automatic retries, and comprehensive observability.

## Features

🚀 **Exactly-Once Execution** - Workflows and steps execute exactly once, even with failures  
⚡ **Durable State** - All state persisted to database, survives crashes and restarts  
🔄 **Automatic Retries** - Configurable retry policies with exponential backoff  
📊 **Rich Observability** - Built-in Prometheus metrics and structured logging  
⏰ **Flexible Scheduling** - Immediate execution, cron schedules, and sleep states  
🏗️ **Distributed Coordination** - Multiple workers with lease-based coordination  
🎯 **Type Safety** - Strongly-typed workflows with compile-time guarantees  
🔧 **Checkpointing** - Automatic step result caching for fault tolerance

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

    "github.com/unkeyed/unkey/go/pkg/clock"
    "github.com/unkeyed/unkey/go/pkg/hydra"
    "github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
    "gorm.io/driver/mysql"
    gormDriver "gorm.io/gorm"
)

// Define your workflow
type OrderWorkflow struct{}

func (w *OrderWorkflow) Name() string {
    return "order-processing"
}

func (w *OrderWorkflow) Run(ctx hydra.WorkflowContext, req *OrderRequest) error {
    // Step 1: Validate payment
    payment, err := hydra.Step(ctx, "validate-payment", func(stepCtx context.Context) (*Payment, error) {
        return validatePayment(stepCtx, req.PaymentID)
    })
    if err != nil {
        return err
    }

    // Step 2: Reserve inventory
    _, err = hydra.Step(ctx, "reserve-inventory", func(stepCtx context.Context) (*Reservation, error) {
        return reserveInventory(stepCtx, req.Items)
    })
    if err != nil {
        return err
    }

    // Step 3: Process order
    _, err = hydra.Step(ctx, "process-order", func(stepCtx context.Context) (*Order, error) {
        return processOrder(stepCtx, payment, req.Items)
    })

    return err
}

func main() {
    // Set up database
    db, err := gormDriver.Open(mysql.Open("dsn"), &gormDriver.Config{})
    if err != nil {
        panic(err)
    }

    // Create store
    store := hydra.NewGORMStore(db, clock.New())

    // Create engine
    engine := hydra.New(hydra.Config{
        Store:     store,
        Namespace: "production",
    })

    // Create worker
    worker, err := hydra.NewWorker(engine, hydra.WorkerConfig{
        WorkerID:    "worker-1",
        Concurrency: 10,
    })
    if err != nil {
        panic(err)
    }

    // Register workflow
    err = hydra.RegisterWorkflow(worker, &OrderWorkflow{})
    if err != nil {
        panic(err)
    }

    // Start worker
    ctx := context.Background()
    err = worker.Start(ctx)
    if err != nil {
        panic(err)
    }
    defer worker.Shutdown(ctx)

    // Submit workflow
    executionID, err := engine.StartWorkflow(ctx, "order-processing", &OrderRequest{
        CustomerID: "cust_123",
        Items:      []Item{{SKU: "item_456", Quantity: 2}},
        PaymentID:  "pay_789",
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Started workflow: %s\n", executionID)
}
```

## Core Concepts

### Engine
The central orchestration component that manages workflow lifecycle and coordinates execution across workers.

```go
engine := hydra.New(hydra.Config{
    Store:     store,
    Namespace: "production",
    Logger:    logger,
})
```

### Workers
Distributed processing units that poll for workflows, acquire leases, and execute workflow logic.

```go
worker, err := hydra.NewWorker(engine, hydra.WorkerConfig{
    WorkerID:          "worker-1",
    Concurrency:       20,
    PollInterval:      100 * time.Millisecond,
    HeartbeatInterval: 30 * time.Second,
    ClaimTimeout:      5 * time.Minute,
})
```

### Workflows
Business logic containers that define a series of steps with exactly-once execution guarantees.

```go
type MyWorkflow struct{}

func (w *MyWorkflow) Name() string { return "my-workflow" }

func (w *MyWorkflow) Run(ctx hydra.WorkflowContext, req *MyRequest) error {
    // Implement your business logic using hydra.Step()
    return nil
}
```

### Steps
Individual units of work with automatic checkpointing and retry logic.

```go
result, err := hydra.Step(ctx, "api-call", func(stepCtx context.Context) (*APIResponse, error) {
    return apiClient.Call(stepCtx, request)
})
```

## Advanced Features

### Sleep States
Suspend workflows for time-based coordination:

```go
// Sleep for 24 hours for manual approval
err = hydra.Sleep(ctx, 24*time.Hour)
if err != nil {
    return err
}

// Continue after sleep
result, err := hydra.Step(ctx, "post-approval", func(stepCtx context.Context) (string, error) {
    return processApprovedRequest(stepCtx)
})
```

### Cron Scheduling
Schedule workflows to run automatically:

```go
err = engine.RegisterCron("0 0 * * *", "daily-report", func(ctx context.Context) error {
    // Generate daily report
    return generateDailyReport(ctx)
})
```

### Error Handling & Retries
Configure retry behavior per workflow:

```go
executionID, err := engine.StartWorkflow(ctx, "order-processing", request,
    hydra.WithMaxAttempts(5),
    hydra.WithRetryBackoff(2*time.Second),
    hydra.WithTimeout(10*time.Minute),
)
```

### Custom Marshallers
Use custom serialization formats:

```go
type ProtobufMarshaller struct{}

func (p *ProtobufMarshaller) Marshal(v any) ([]byte, error) {
    // Implement protobuf marshalling
}

func (p *ProtobufMarshaller) Unmarshal(data []byte, v any) error {
    // Implement protobuf unmarshalling
}

engine := hydra.New(hydra.Config{
    Store:      store,
    Marshaller: &ProtobufMarshaller{},
})
```

## Observability

### Prometheus Metrics

Hydra provides comprehensive metrics out of the box:

**Workflow Metrics:**
- `hydra_workflows_started_total` - Total workflows started
- `hydra_workflows_completed_total` - Total workflows completed/failed
- `hydra_workflow_duration_seconds` - Workflow execution time
- `hydra_workflow_queue_time_seconds` - Time spent waiting for execution
- `hydra_workflows_active` - Currently running workflows per worker

**Step Metrics:**
- `hydra_steps_executed_total` - Total steps executed with status
- `hydra_step_duration_seconds` - Individual step execution time
- `hydra_steps_cached_total` - Steps served from checkpoint cache
- `hydra_steps_retried_total` - Step retry attempts

**Worker Metrics:**
- `hydra_worker_polls_total` - Worker polling operations
- `hydra_worker_heartbeats_total` - Worker heartbeat operations
- `hydra_lease_acquisitions_total` - Workflow lease acquisitions
- `hydra_worker_concurrency_current` - Current workflow concurrency per worker

### Example Grafana Queries

```promql
# Workflow throughput
rate(hydra_workflows_completed_total[5m])

# Average workflow duration
rate(hydra_workflow_duration_seconds_sum[5m]) / rate(hydra_workflow_duration_seconds_count[5m])

# Step cache hit rate
rate(hydra_steps_cached_total[5m]) / rate(hydra_steps_executed_total[5m])

# Worker utilization
hydra_workflows_active / hydra_worker_concurrency_current
```

## Architecture

Hydra uses a lease-based coordination model for distributed execution:

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Worker 1  │    │   Worker 2  │    │   Worker N  │
│             │    │             │    │             │
│ ┌─────────┐ │    │ ┌─────────┐ │    │ ┌─────────┐ │
│ │ Poll    │ │    │ │ Poll    │ │    │ │ Poll    │ │
│ │ Execute │ │    │ │ Execute │ │    │ │ Execute │ │
│ │ Heartbeat│ │    │ │ Heartbeat│ │    │ │ Heartbeat│ │
│ └─────────┘ │    │ └─────────┘ │    │ └─────────┘ │
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                  ┌─────────────────┐
                  │    Database     │
                  │                 │
                  │ • Workflows     │
                  │ • Steps         │
                  │ • Leases        │
                  │ • Cron Jobs     │
                  └─────────────────┘
```

1. **Workers poll** the database for pending workflows
2. **Workers acquire leases** on available workflows for exclusive execution
3. **Workers execute** workflow logic with step-by-step checkpointing
4. **Workers send heartbeats** to maintain lease ownership
5. **Completed workflows** update status and release leases

## Database Schema

Hydra requires the following tables (auto-migrated with GORM):

```sql
-- Workflow executions
CREATE TABLE workflow_executions (
    id VARCHAR(255) PRIMARY KEY,
    workflow_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    input_data LONGBLOB,
    output_data LONGBLOB,
    error_message TEXT,
    max_attempts INT NOT NULL,
    remaining_attempts INT NOT NULL,
    created_at BIGINT NOT NULL,
    started_at BIGINT,
    completed_at BIGINT,
    trigger_type VARCHAR(50),
    trigger_source VARCHAR(255),
    INDEX idx_workflow_executions_status_namespace (status, namespace),
    INDEX idx_workflow_executions_workflow_name (workflow_name)
);

-- Workflow steps
CREATE TABLE workflow_steps (
    id VARCHAR(255) PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    step_order INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    input_data LONGBLOB,
    output_data LONGBLOB,
    error_message TEXT,
    max_attempts INT NOT NULL,
    remaining_attempts INT NOT NULL,
    started_at BIGINT,
    completed_at BIGINT,
    UNIQUE KEY unique_execution_step (execution_id, step_name),
    INDEX idx_workflow_steps_execution_id (execution_id)
);

-- Leases for coordination
CREATE TABLE leases (
    resource_id VARCHAR(255) PRIMARY KEY,
    kind VARCHAR(50) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    worker_id VARCHAR(255) NOT NULL,
    acquired_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL,
    heartbeat_at BIGINT NOT NULL,
    INDEX idx_leases_expires_at (expires_at),
    INDEX idx_leases_worker_id (worker_id)
);

-- Cron jobs
CREATE TABLE cron_jobs (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    cron_spec VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    workflow_name VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    next_run_at BIGINT NOT NULL,
    UNIQUE KEY unique_namespace_name (namespace, name),
    INDEX idx_cron_jobs_next_run_at (next_run_at, enabled)
);
```

## Performance Considerations

### Scaling Workers
- **Horizontal scaling**: Add more worker instances
- **Vertical scaling**: Increase concurrency per worker
- **Database optimization**: Ensure proper indexing and connection pooling

### Optimizing Workflows
- **Idempotent steps**: Ensure steps can be safely retried
- **Minimize step payload size**: Reduce serialization overhead
- **Batch operations**: Combine multiple operations in single steps
- **Use appropriate timeouts**: Balance responsiveness vs. reliability

### Database Tuning
```sql
-- Recommended indexes for performance
CREATE INDEX idx_workflow_executions_polling 
ON workflow_executions (status, namespace, created_at);

CREATE INDEX idx_leases_cleanup 
ON leases (expires_at);

CREATE INDEX idx_workflow_steps_execution_order 
ON workflow_steps (execution_id, step_order);
```

## Best Practices

### Workflow Design
- ✅ **Keep workflows stateless** - Store state in steps, not workflow instances
- ✅ **Make steps idempotent** - Steps should be safe to retry
- ✅ **Use descriptive step names** - Names should be stable across deployments
- ✅ **Handle errors gracefully** - Distinguish between retryable and permanent errors
- ✅ **Minimize external dependencies** - Use timeouts and circuit breakers

### Production Deployment
- ✅ **Monitor metrics** - Set up alerts for error rates and latency
- ✅ **Configure retries** - Set appropriate retry policies for your use case
- ✅ **Database backup** - Ensure workflow state is backed up
- ✅ **Graceful shutdown** - Handle SIGTERM to finish active workflows
- ✅ **Resource limits** - Set memory and CPU limits for workers

## Examples

### Order Processing Workflow
```go
type OrderWorkflow struct {
    paymentService   PaymentService
    inventoryService InventoryService
    shippingService  ShippingService
}

func (w *OrderWorkflow) Run(ctx hydra.WorkflowContext, req *OrderRequest) error {
    // Validate and charge payment
    payment, err := hydra.Step(ctx, "process-payment", func(stepCtx context.Context) (*Payment, error) {
        return w.paymentService.ProcessPayment(stepCtx, &PaymentRequest{
            Amount:   req.TotalAmount,
            Method:   req.PaymentMethod,
            Customer: req.CustomerID,
        })
    })
    if err != nil {
        return err
    }

    // Reserve inventory
    reservation, err := hydra.Step(ctx, "reserve-inventory", func(stepCtx context.Context) (*Reservation, error) {
        return w.inventoryService.ReserveItems(stepCtx, req.Items)
    })
    if err != nil {
        // Refund payment on inventory failure
        hydra.Step(ctx, "refund-payment", func(stepCtx context.Context) (any, error) {
            return nil, w.paymentService.RefundPayment(stepCtx, payment.ID)
        })
        return err
    }

    // Create shipping label
    _, err = hydra.Step(ctx, "create-shipping", func(stepCtx context.Context) (*ShippingLabel, error) {
        return w.shippingService.CreateLabel(stepCtx, &ShippingRequest{
            Address:     req.ShippingAddress,
            Items:       req.Items,
            Reservation: reservation.ID,
        })
    })

    return err
}
```

### Approval Workflow with Sleep
```go
func (w *ApprovalWorkflow) Run(ctx hydra.WorkflowContext, req *ApprovalRequest) error {
    // Submit for review
    _, err := hydra.Step(ctx, "submit-review", func(stepCtx context.Context) (*Review, error) {
        return w.reviewService.SubmitForReview(stepCtx, req)
    })
    if err != nil {
        return err
    }

    // Sleep for 48 hours to allow manual review
    err = hydra.Sleep(ctx, 48*time.Hour)
    if err != nil {
        return err
    }

    // Check approval status
    approval, err := hydra.Step(ctx, "check-approval", func(stepCtx context.Context) (*Approval, error) {
        return w.reviewService.GetApprovalStatus(stepCtx, req.ID)
    })
    if err != nil {
        return err
    }

    if approval.Status == "approved" {
        // Process approved request
        _, err = hydra.Step(ctx, "process-approved", func(stepCtx context.Context) (any, error) {
            return nil, w.processApprovedRequest(stepCtx, req)
        })
    }

    return err
}
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](../../CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.

---

**Need help?** Check out our [documentation](https://docs.unkey.com) or join our [Discord community](https://discord.gg/unkey).