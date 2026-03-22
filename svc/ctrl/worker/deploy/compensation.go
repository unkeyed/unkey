package deploy

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	restate "github.com/restatedev/sdk-go"
)

// Compensation stores deployment rollback actions.
//
// Actions are registered in forward order with [Compensation.Add] and executed
// in reverse order by [Compensation.Execute], so teardown naturally unwinds the
// setup sequence that succeeded before a failure.
type Compensation struct {
	mu         sync.Mutex
	operations []func(ctx restate.ObjectContext) error
}

// NewCompensation creates an empty [Compensation].
func NewCompensation() *Compensation {
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
