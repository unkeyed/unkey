package harness

import (
	"context"
	"slices"
	"sync"

	"github.com/unkeyed/unkey/pkg/workos"
	"github.com/unkeyed/unkey/svc/ctrl/worker/gdprerasure"
)

// WithoutStripe binds GdprErasureWorkflow without a Stripe client, as a worker
// with no Stripe key runs it.
func WithoutStripe() Option {
	return func(o *harnessOpts) {
		o.withoutStripe = true
	}
}

// ErasedRecords records the external deletes GdprErasureWorkflow requests.
// Each delete reports whether the record existed the first time its id is
// deleted, as the real services do.
type ErasedRecords struct {
	mu                sync.Mutex
	stripeCustomerIDs []string
	workOSUserIDs     []string
	workOSOrgIDs      []string
}

var (
	_ gdprerasure.StripeCustomers = (*ErasedRecords)(nil)
	_ workos.Eraser               = (*ErasedRecords)(nil)
)

func (e *ErasedRecords) DeleteCustomer(_ context.Context, customerID string) (bool, error) {
	return e.record(&e.stripeCustomerIDs, customerID), nil
}

func (e *ErasedRecords) DeleteUser(_ context.Context, userID string) (bool, error) {
	return e.record(&e.workOSUserIDs, userID), nil
}

func (e *ErasedRecords) DeleteOrganization(_ context.Context, orgID string) (bool, error) {
	return e.record(&e.workOSOrgIDs, orgID), nil
}

// StripeCustomerIDs returns the Stripe customers deleted so far.
func (e *ErasedRecords) StripeCustomerIDs() []string { return e.snapshot(e.stripeCustomerIDs) }

// WorkOSUserIDs returns the WorkOS users deleted so far.
func (e *ErasedRecords) WorkOSUserIDs() []string { return e.snapshot(e.workOSUserIDs) }

// WorkOSOrgIDs returns the WorkOS organizations deleted so far.
func (e *ErasedRecords) WorkOSOrgIDs() []string { return e.snapshot(e.workOSOrgIDs) }

func (e *ErasedRecords) record(ids *[]string, id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if slices.Contains(*ids, id) {
		return false
	}
	*ids = append(*ids, id)
	return true
}

func (e *ErasedRecords) snapshot(ids []string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(ids)
}
