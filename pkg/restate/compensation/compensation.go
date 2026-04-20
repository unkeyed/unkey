// Package compensation provides a rollback mechanism for multi-step Restate workflows.
//
// When a workflow performs a sequence of side effects (database writes, API calls,
// resource provisioning), a failure partway through leaves the system in a partially
// mutated state. [Compensation] collects undo actions as the workflow progresses and
// executes them in reverse order on failure, unwinding the setup sequence.
//
// This follows the saga compensation pattern: each forward step registers its
// inverse, and the inverse runs only if the workflow fails after that step.
//
// # Usage
//
//	comp := compensation.New()
//
//	comp.Add("delete topology", func(ctx restate.RunContext) error {
//	    return db.Query.DeleteTopology(ctx, tx, topologyID)
//	})
//
//	// ... more steps, each registering their undo ...
//
//	if err != nil {
//	    return errors.Join(err, comp.Execute(ctx))
//	}
package compensation

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	restate "github.com/restatedev/sdk-go"
)

// Compensation stores rollback actions for Restate workflows.
//
// Actions are registered in forward order with [Compensation.Add] and executed
// in reverse order by [Compensation.Execute], so teardown naturally unwinds the
// setup sequence that succeeded before a failure.
type Compensation struct {
	mu         sync.Mutex
	operations []func(ctx restate.ObjectContext) error
}

// New creates an empty [Compensation].
func New() *Compensation {
	return &Compensation{
		mu:         sync.Mutex{},
		operations: []func(ctx restate.ObjectContext) error{},
	}
}

// Add registers a compensation step.
//
// Add wraps run in [restate.RunVoid] and stores it for later execution during
// [Compensation.Execute]. The step name is used as Restate run metadata.
func (c *Compensation) Add(name string, run func(ctx restate.RunContext) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.operations = append(c.operations, func(ctx restate.ObjectContext) error {
		return restate.RunVoid(ctx, run, restate.WithName(fmt.Sprintf("[compensation]: %s", name)))
	})
}

// AddCtx registers a compensation step that needs the full ObjectContext
// (e.g. to Send messages to other Restate services). Unlike [Add], the
// callback is NOT wrapped in [restate.RunVoid] — the caller is responsible
// for making the operation idempotent. Use this for fire-and-forget Sends
// like releasing a BuildSlotService slot.
func (c *Compensation) AddCtx(run func(ctx restate.ObjectContext) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.operations = append(c.operations, run)
}

// Execute runs all registered compensations in reverse registration order.
//
// Execute returns nil when every step succeeds. When multiple steps fail, it
// joins all failures with [errors.Join]. Execute does not clear registered
// steps, so calling it more than once re-runs the same compensations.
func (c *Compensation) Execute(ctx restate.ObjectContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var errs error

	for _, op := range slices.Backward(c.operations) {
		if err := op(ctx); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}
