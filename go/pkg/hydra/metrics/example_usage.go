package metrics

import (
	"time"
)

const exampleNamespace = "production"

func ExampleWorkflowMetrics() {
	namespace := exampleNamespace
	workflowName := "user-onboarding"

	WorkflowsStartedTotal.WithLabelValues(namespace, workflowName, "manual").Inc()

	WorkflowsQueued.WithLabelValues(namespace, "pending").Set(42)
	WorkflowsActive.WithLabelValues(namespace, "worker-1").Set(5)

	start := time.Now()
	ObserveWorkflowDuration(namespace, workflowName, "completed", start)
	WorkflowsCompletedTotal.WithLabelValues(namespace, workflowName, "completed").Inc()
}

func ExampleStepMetrics() {
	namespace := exampleNamespace
	workflowName := "order-processing"
	stepName := "charge-payment"

	start := time.Now()
	ObserveStepDuration(namespace, workflowName, stepName, "completed", start)
	StepsExecutedTotal.WithLabelValues(namespace, workflowName, stepName, "completed").Inc()

	StepsCachedTotal.WithLabelValues(namespace, workflowName, stepName).Inc()
}

func ExampleDatabaseMetrics() {
	start := time.Now()
	ObserveDbOperation("select", "workflow_executions", "success", start)

	DbConnectionsActive.WithLabelValues("worker-1").Set(15)
}

func ExampleSleepMetrics() {
	namespace := exampleNamespace
	workflowName := "user-onboarding"

	SleepsStartedTotal.WithLabelValues(namespace, workflowName).Inc()
	SleepsResumedTotal.WithLabelValues(namespace, workflowName).Inc()

	actualSleepDuration := 25 * time.Minute // actual time slept
	SleepDurationSeconds.WithLabelValues(namespace, workflowName).Observe(actualSleepDuration.Seconds())

	CronTriggersTotal.WithLabelValues(namespace, "daily-report", "success").Inc()
}

func ExampleErrorMetrics() {
	namespace := exampleNamespace

	RecordError(namespace, "step", "timeout")
	RecordError(namespace, "client", "serialization")
	RecordError(namespace, "store", "connection")

	PanicsTotal.WithLabelValues("worker-1", "step_execution").Inc()

	TimeoutsTotal.WithLabelValues(namespace, "workflow_execution").Inc()
}

func ExamplePayloadMetrics() {
	namespace := exampleNamespace
	workflowName := "image-processing"

	inputSize := 1024 * 50 // 50KB input
	outputSize := 1024 * 5 // 5KB output

	RecordPayloadSize(namespace, workflowName, "input", inputSize)
	RecordPayloadSize(namespace, workflowName, "output", outputSize)

	SerializationErrorsTotal.WithLabelValues(namespace, workflowName, "input").Inc()
}

func ExampleWorkerMetrics() {
	workerID := "worker-1"
	namespace := exampleNamespace

	WorkerHeartbeatsTotal.WithLabelValues(workerID, namespace, "success").Inc()
	WorkerPollsTotal.WithLabelValues(workerID, namespace, "found_work").Inc()
	LeaseAcquisitionsTotal.WithLabelValues(workerID, "workflow", "success").Inc()

	WorkerConcurrencyCurrent.WithLabelValues(workerID, namespace).Set(8)
}
