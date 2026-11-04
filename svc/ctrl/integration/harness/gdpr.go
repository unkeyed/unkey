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

// WithoutResend binds GdprErasureWorkflow without a Resend client, as a worker
// with no Resend key runs it.
func WithoutResend() Option {
	return func(o *harnessOpts) {
		o.withoutResend = true
	}
}

// WithoutPlain binds GdprErasureWorkflow without a Plain client, as a worker
// with no Plain key runs it.
func WithoutPlain() Option {
	return func(o *harnessOpts) {
		o.withoutPlain = true
	}
}

// WithoutDepot binds GdprErasureWorkflow without a Depot client, as a worker
// that does not build on Depot runs it.
func WithoutDepot() Option {
	return func(o *harnessOpts) {
		o.withoutDepot = true
	}
}

// ErasedRecords records the external deletes GdprErasureWorkflow requests.
// Each delete reports whether the record existed the first time its id is
// deleted, as the real services do. A WorkOS user added with AddWorkOSUser
// has an email until it is deleted, and a Plain customer added with
// AddPlainCustomer can be found until it is deleted.
type ErasedRecords struct {
	mu                  sync.Mutex
	stripeCustomerIDs   []string
	workOSUserEmails    map[string]string
	workOSUserIDs       []string
	workOSOrgIDs        []string
	resendContactEmails []string
	plainCustomers      []plainCustomer
	plainCustomerIDs    []string
	depotProjectIDs     []string
}

type plainCustomer struct {
	id         string
	email      string
	externalID string
}

var (
	_ gdprerasure.StripeCustomers = (*ErasedRecords)(nil)
	_ gdprerasure.ResendContacts  = (*ErasedRecords)(nil)
	_ gdprerasure.PlainCustomers  = (*ErasedRecords)(nil)
	_ gdprerasure.DepotProjects   = (*ErasedRecords)(nil)
	_ workos.Eraser               = (*ErasedRecords)(nil)
)

// AddWorkOSUser makes userID a WorkOS user with this email address.
func (e *ErasedRecords) AddWorkOSUser(userID, email string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.workOSUserEmails == nil {
		e.workOSUserEmails = make(map[string]string)
	}
	e.workOSUserEmails[userID] = email
}

func (e *ErasedRecords) UserEmail(_ context.Context, userID string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if slices.Contains(e.workOSUserIDs, userID) {
		return "", nil
	}
	return e.workOSUserEmails[userID], nil
}

// AddPlainCustomer makes customerID a Plain customer with this email address
// and external id.
func (e *ErasedRecords) AddPlainCustomer(customerID, email, externalID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.plainCustomers = append(e.plainCustomers, plainCustomer{id: customerID, email: email, externalID: externalID})
}

func (e *ErasedRecords) DeleteCustomers(_ context.Context, email, externalID string) (int64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var deleted int64
	e.plainCustomers = slices.DeleteFunc(e.plainCustomers, func(c plainCustomer) bool {
		if (email == "" || c.email != email) && (externalID == "" || c.externalID != externalID) {
			return false
		}
		e.plainCustomerIDs = append(e.plainCustomerIDs, c.id)
		deleted++
		return true
	})
	return deleted, nil
}

func (e *ErasedRecords) DeleteProject(_ context.Context, projectID string) (bool, error) {
	return e.record(&e.depotProjectIDs, projectID), nil
}

func (e *ErasedRecords) DeleteContact(_ context.Context, email string) (bool, error) {
	return e.record(&e.resendContactEmails, email), nil
}

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

// ResendContactEmails returns the Resend contacts deleted so far.
func (e *ErasedRecords) ResendContactEmails() []string { return e.snapshot(e.resendContactEmails) }

// PlainCustomerIDs returns the Plain customers deleted so far.
func (e *ErasedRecords) PlainCustomerIDs() []string { return e.snapshot(e.plainCustomerIDs) }

// DepotProjectIDs returns the Depot projects deleted so far.
func (e *ErasedRecords) DepotProjectIDs() []string { return e.snapshot(e.depotProjectIDs) }

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
