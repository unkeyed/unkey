package k8s

import (
	"context"

	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Reconciler defines the interface for Kubernetes reconciliation operations.
//
// This interface mirrors controller-runtime's reconcile interface but is defined
// within the k8s package to provide a common abstraction for
// different types of reconcilers. All reconcilers in krane implement
// this interface to ensure consistent error handling and result patterns.
//
// The interface expects implementations to:
//   - Handle context cancellation gracefully
//   - Return appropriate requeue timing for reconciliation
//   - Provide meaningful error messages for debugging
type Reconciler interface {
	Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error)
}
