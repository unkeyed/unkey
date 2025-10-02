// Package hydra provides a distributed workflow orchestration engine designed
// for reliable execution of multi-step business processes at scale.
//
// Hydra implements the Temporal-style workflow pattern with durable execution,
// automatic retries, checkpointing, and distributed coordination. It supports
// both simple sequential workflows and complex long-running processes with
// sleep states, cron scheduling, and step-level fault tolerance.
//
// # Core Concepts
//
// Engine: The central orchestration component that manages workflow lifecycle,
// worker coordination, and persistence. Each engine instance operates within
// a specific namespace for tenant isolation.
//
// Workers: Distributed processing units that poll for pending workflows,
// acquire leases for exclusive execution, and run workflow logic. Workers
// support concurrent execution with configurable limits and automatic
// heartbeat management.
//
// Workflows: Business logic containers that define a series of steps to be
// executed. Workflows are stateless functions that can be suspended, resumed,
// and retried while maintaining exactly-once execution guarantees.
//
// Steps: Individual units of work within a workflow. Steps support automatic
// checkpointing, retry logic, and result caching to ensure idempotent execution
// even across worker failures or restarts.
//
// # Key Features
//
// Exactly-Once Execution: Workflows and steps execute exactly once, even in
// the presence of worker failures, network partitions, or duplicate deliveries.
//
// Durable State: All workflow state is persisted to a database, allowing
// workflows to survive process restarts and infrastructure failures.
//
// Distributed Coordination: Multiple workers can safely operate on the same
// workflow queue using lease-based coordination and circuit breaker protection.
//
// Comprehensive Observability: Built-in Prometheus metrics track workflow
// throughput, latency, error rates, and system health across all components.
//
// Flexible Scheduling: Support for immediate execution, cron-based scheduling,
// and workflow sleep states for time-based coordination.
//
// # Basic Usage
//
// Creating an engine and worker:
//
//	// Create the engine with database DSN
//	engine, err := hydra.NewEngine(hydra.Config{
//	    DSN:       "user:password@tcp(localhost:3306)/hydra",
//	    Namespace: "production",
//	    Logger:    logger,
//	})
//	if err != nil {
//	    return err
//	}
//
//	// Create and configure a worker
//	worker, err := hydra.NewWorker(engine, hydra.WorkerConfig{
//	    WorkerID:          "worker-1",
//	    Concurrency:       10,
//	    PollInterval:      100 * time.Millisecond,
//	    HeartbeatInterval: 30 * time.Second,
//	    ClaimTimeout:      5 * time.Minute,
//	})
//
// Defining a workflow:
//
//	type OrderWorkflow struct {
//	    engine *hydra.Engine
//	}
//
//	func (w *OrderWorkflow) Name() string {
//	    return "order-processing"
//	}
//
//	func (w *OrderWorkflow) Run(ctx hydra.WorkflowContext, req *OrderRequest) error {
//	    // Step 1: Validate payment
//	    payment, err := hydra.Step(ctx, "validate-payment", func(stepCtx context.Context) (*Payment, error) {
//	        return validatePayment(stepCtx, req.PaymentID)
//	    })
//	    if err != nil {
//	        return err
//	    }
//
//	    // Step 2: Reserve inventory
//	    reservation, err := hydra.Step(ctx, "reserve-inventory", func(stepCtx context.Context) (*Reservation, error) {
//	        return reserveInventory(stepCtx, req.Items)
//	    })
//	    if err != nil {
//	        return err
//	    }
//
//	    // Step 3: Process order
//	    _, err = hydra.Step(ctx, "process-order", func(stepCtx context.Context) (*Order, error) {
//	        return processOrder(stepCtx, payment, reservation)
//	    })
//
//	    return err
//	}
//
// Starting workflows:
//
//	// Register the workflow with the worker
//	orderWorkflow := &OrderWorkflow{engine: engine}
//	err = hydra.RegisterWorkflow(worker, orderWorkflow)
//	if err != nil {
//	    return err
//	}
//
//	// Start the worker
//	ctx := context.Background()
//	err = worker.Start(ctx)
//	if err != nil {
//	    return err
//	}
//	defer worker.Shutdown(ctx)
//
//	// Submit a workflow for execution
//	request := &OrderRequest{
//	    CustomerID: "cust_123",
//	    Items:      []Item{{SKU: "item_456", Quantity: 2}},
//	    PaymentID:  "pay_789",
//	}
//
//	executionID, err := engine.StartWorkflow(ctx, "order-processing", request)
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Started workflow execution: %s\n", executionID)
//
// # Marshalling Options
//
// Hydra supports multiple marshalling formats for workflow payloads and step results:
//
// JSON Marshaller (Default):
//
//	engine, err := hydra.NewEngine(hydra.Config{
//	    Marshaller: hydra.NewJSONMarshaller(), // Default if not specified
//	    // ... other config
//	})
//
// # Advanced Features
//
// Sleep States: Workflows can suspend execution and resume after a specified
// duration, allowing for time-based coordination and human approval processes:
//
//	// Sleep for 24 hours for manual approval
//	return hydra.Sleep(ctx, 24*time.Hour)
//
// Cron Scheduling: Register workflows to run on a schedule:
//
//	err = engine.RegisterCron("0 0 * * *", "daily-report", func(ctx context.Context) error {
//	    // Generate daily report
//	    return generateDailyReport(ctx)
//	})
//
// Error Handling and Retries: Configure retry behavior at the workflow level:
//
//	executionID, err := engine.StartWorkflow(ctx, "order-processing", request,
//	    hydra.WithMaxAttempts(5),
//	    hydra.WithRetryBackoff(2*time.Second),
//	    hydra.WithTimeout(10*time.Minute),
//	)
//
// # Observability
//
// Hydra provides comprehensive Prometheus metrics out of the box:
//
// Workflow Metrics:
// - hydra_workflows_started_total: Total workflows started
// - hydra_workflows_completed_total: Total workflows completed/failed
// - hydra_workflow_duration_seconds: Workflow execution time
// - hydra_workflow_queue_time_seconds: Time spent waiting for execution
// - hydra_workflows_active: Currently running workflows per worker
//
// Step Metrics:
// - hydra_steps_executed_total: Total steps executed with status
// - hydra_step_duration_seconds: Individual step execution time
// - hydra_steps_cached_total: Steps served from checkpoint cache
// - hydra_steps_retried_total: Step retry attempts
//
// Worker Metrics:
// - hydra_worker_polls_total: Worker polling operations
// - hydra_worker_heartbeats_total: Worker heartbeat operations
// - hydra_lease_acquisitions_total: Workflow lease acquisitions
// - hydra_worker_concurrency_current: Current workflow concurrency per worker
//
// Error and Performance Metrics:
// - hydra_errors_total: Categorized error counts
// - hydra_payload_size_bytes: Workflow and step payload sizes
// - hydra_db_operations_total: Database operation counts and latency
//
// All metrics include rich labels for namespace, workflow names, worker IDs,
// and status information, enabling detailed monitoring and alerting.
//
// # Architecture
//
// Hydra uses a lease-based coordination model to ensure exactly-once execution:
//
// 1. Workers poll the database for pending workflows in their namespace
// 2. Workers attempt to acquire exclusive leases on available workflows
// 3. Successful lease holders execute the workflow logic
// 4. Workers send periodic heartbeats to maintain lease ownership
// 5. Completed workflows update their status and release the lease
// 6. Failed workers automatically lose their leases after timeout
//
// This design provides fault tolerance without requiring complex consensus
// protocols or external coordination services.
//
// # Database Schema
//
// Hydra requires the following database tables:
//
// - workflow_executions: Stores workflow state, status, and metadata
// - workflow_steps: Tracks individual step execution and results
// - leases: Manages worker coordination and exclusive access
// - cron_jobs: Stores scheduled workflow definitions
//
// The schema should be created using the provided schema.sql file.
//
// # Error Handling
//
// Hydra distinguishes between different types of errors:
//
// Transient Errors: Network timeouts, temporary database failures, etc.
// These trigger automatic retries based on the configured retry policy.
//
// Permanent Errors: Validation failures, business logic errors, etc.
// These immediately fail the workflow without retries.
//
// Workflow Suspension: Controlled suspension using Sleep() for time-based
// coordination or external event waiting.
//
// # Performance Considerations
//
// - Workers use circuit breakers to prevent cascading failures
// - Database queries are optimized with appropriate indexes
// - Lease timeouts prevent stuck workflows from blocking execution
// - Configurable concurrency limits prevent resource exhaustion
// - Built-in connection pooling and retry logic for database operations
//
// # Thread Safety
//
// All Hydra components are thread-safe and designed for concurrent access:
// - Multiple workers can safely operate on the same workflow queue
// - Step execution is atomic and isolated using database transactions
// - Workflow state updates use optimistic locking to prevent race conditions
// - Metrics collection is thread-safe and non-blocking
package hydra
