package hydra

import (
	"context"
	"testing"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
	"github.com/stretchr/testify/require"
)

// Store unit tests

func newTestStore(t *testing.T) store.Store {
	s, err := gorm.NewSQLiteStore("", nil)
	require.NoError(t, err)
	return s
}

func TestGetPendingWorkflows(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	
	// Given: A pending workflow in the database
	workflow := &WorkflowExecution{
		ID:           "wf_test123",
		WorkflowName: "test-workflow",
		Status:       WorkflowStatusPending,
		Namespace:    "default",
		CreatedAt:    time.Now().UnixMilli(),
		InputData:    []byte(`{}`),
		MaxAttempts:  3,
		RemainingAttempts: 3,
	}
	
	err := store.CreateWorkflow(ctx, workflow)
	require.NoError(t, err)
	
	// When: Getting pending workflows
	pending, err := store.GetPendingWorkflows(ctx, "default", 10, []string{"test-workflow"})
	require.NoError(t, err)
	
	// Then: Should find the pending workflow
	require.Len(t, pending, 1, "Should find 1 pending workflow")
	require.Equal(t, "wf_test123", pending[0].ID)
	require.Equal(t, "test-workflow", pending[0].WorkflowName)
	require.Equal(t, WorkflowStatusPending, pending[0].Status)
}

func TestGetPendingWorkflows_FiltersByWorkflowName(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	
	// Given: Two pending workflows with different names
	workflow1 := &WorkflowExecution{
		ID:           "wf_test1",
		WorkflowName: "workflow-a",
		Status:       WorkflowStatusPending,
		Namespace:    "default",
		CreatedAt:    time.Now().UnixMilli(),
		InputData:    []byte(`{}`),
		MaxAttempts:  3,
		RemainingAttempts: 3,
	}
	
	workflow2 := &WorkflowExecution{
		ID:           "wf_test2", 
		WorkflowName: "workflow-b",
		Status:       WorkflowStatusPending,
		Namespace:    "default",
		CreatedAt:    time.Now().UnixMilli(),
		InputData:    []byte(`{}`),
		MaxAttempts:  3,
		RemainingAttempts: 3,
	}
	
	require.NoError(t, store.CreateWorkflow(ctx, workflow1))
	require.NoError(t, store.CreateWorkflow(ctx, workflow2))
	
	// When: Getting pending workflows for only "workflow-a"
	pending, err := store.GetPendingWorkflows(ctx, "default", 10, []string{"workflow-a"})
	require.NoError(t, err)
	
	// Then: Should only find workflow-a
	require.Len(t, pending, 1, "Should find only 1 workflow")
	require.Equal(t, "workflow-a", pending[0].WorkflowName)
}

func TestGetPendingWorkflows_ExcludesNonPendingWorkflows(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	
	// Given: Workflows in different states
	pendingWorkflow := &WorkflowExecution{
		ID:           "wf_pending",
		WorkflowName: "test-workflow",
		Status:       WorkflowStatusPending,
		Namespace:    "default",
		CreatedAt:    time.Now().UnixMilli(),
		InputData:    []byte(`{}`),
		MaxAttempts:  3,
		RemainingAttempts: 3,
	}
	
	completedWorkflow := &WorkflowExecution{
		ID:           "wf_completed",
		WorkflowName: "test-workflow", 
		Status:       WorkflowStatusCompleted,
		Namespace:    "default",
		CreatedAt:    time.Now().UnixMilli(),
		InputData:    []byte(`{}`),
		MaxAttempts:  3,
		RemainingAttempts: 3,
	}
	
	require.NoError(t, store.CreateWorkflow(ctx, pendingWorkflow))
	require.NoError(t, store.CreateWorkflow(ctx, completedWorkflow))
	
	// When: Getting pending workflows
	pending, err := store.GetPendingWorkflows(ctx, "default", 10, []string{"test-workflow"})
	require.NoError(t, err)
	
	// Then: Should only find the pending workflow
	require.Len(t, pending, 1, "Should find only pending workflow")
	require.Equal(t, "wf_pending", pending[0].ID)
}

func TestAcquireWorkflowLease_NewLease(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	
	// Given: A pending workflow and no existing lease
	workflowID := "wf_test123"
	workerID := "worker-123"
	leaseDuration := 5 * time.Minute
	
	// Create a pending workflow first
	workflow := &WorkflowExecution{
		ID:           workflowID,
		WorkflowName: "test-workflow",
		Status:       WorkflowStatusPending,
		Namespace:    "default",
		CreatedAt:    time.Now().UnixMilli(),
		InputData:    []byte(`{}`),
		MaxAttempts:  3,
		RemainingAttempts: 3,
	}
	err := store.CreateWorkflow(ctx, workflow)
	require.NoError(t, err)
	
	// When: Acquiring lease
	err = store.AcquireWorkflowLease(ctx, workflowID, "default", workerID, leaseDuration)
	
	// Then: Should succeed
	require.NoError(t, err)
	
	// And: Lease should be created
	lease, err := store.GetLease(ctx, workflowID)
	require.NoError(t, err)
	require.Equal(t, workflowID, lease.ResourceID)
	require.Equal(t, "workflow", lease.Kind)
	require.Equal(t, workerID, lease.WorkerID)
	require.Equal(t, "default", lease.Namespace)
}

func TestAcquireWorkflowLease_ExistingExpiredLease(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	
	// Given: A pending workflow and an expired lease exists
	workflowID := "wf_test123"
	oldWorkerID := "worker-old"
	newWorkerID := "worker-new"
	
	// Create a pending workflow first
	workflow := &WorkflowExecution{
		ID:           workflowID,
		WorkflowName: "test-workflow",
		Status:       WorkflowStatusPending,
		Namespace:    "default",
		CreatedAt:    time.Now().UnixMilli(),
		InputData:    []byte(`{}`),
		MaxAttempts:  3,
		RemainingAttempts: 3,
	}
	err := store.CreateWorkflow(ctx, workflow)
	require.NoError(t, err)
	
	// Create expired lease
	expiredLease := &Lease{
		ResourceID:  workflowID,
		Kind:        "workflow",
		Namespace:   "default",
		WorkerID:    oldWorkerID,
		AcquiredAt:  time.Now().Add(-10*time.Minute).UnixMilli(),
		ExpiresAt:   time.Now().Add(-5*time.Minute).UnixMilli(), // Expired
		HeartbeatAt: time.Now().Add(-10*time.Minute).UnixMilli(),
	}
	
	err = store.AcquireLease(ctx, expiredLease)
	require.NoError(t, err)
	
	// When: New worker tries to acquire lease
	err = store.AcquireWorkflowLease(ctx, workflowID, "default", newWorkerID, 5*time.Minute)
	
	// Then: Should succeed (take over expired lease)
	require.NoError(t, err)
	
	// And: Lease should be owned by new worker
	lease, err := store.GetLease(ctx, workflowID)
	require.NoError(t, err)
	require.Equal(t, newWorkerID, lease.WorkerID)
}

func TestAcquireWorkflowLease_ExistingActiveLease(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	
	// Given: A pending workflow exists
	workflowID := "wf_test123"
	workerID1 := "worker-1"
	workerID2 := "worker-2"
	
	// Create a pending workflow first
	workflow := &WorkflowExecution{
		ID:           workflowID,
		WorkflowName: "test-workflow",
		Status:       WorkflowStatusPending,
		Namespace:    "default",
		CreatedAt:    time.Now().UnixMilli(),
		InputData:    []byte(`{}`),
		MaxAttempts:  3,
		RemainingAttempts: 3,
	}
	err := store.CreateWorkflow(ctx, workflow)
	require.NoError(t, err)
	
	// Worker 1 acquires lease
	err = store.AcquireWorkflowLease(ctx, workflowID, "default", workerID1, 5*time.Minute)
	require.NoError(t, err)
	
	// When: Worker 2 tries to acquire same lease
	err = store.AcquireWorkflowLease(ctx, workflowID, "default", workerID2, 5*time.Minute)
	
	// Then: Should fail
	require.Error(t, err)
	require.Contains(t, err.Error(), "already leased")
}