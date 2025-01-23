package fault

import (
	"errors"
	"slices"
)

// Step represents a single step in an error chain, containing both
// the error message and the location where it occurred.
type Step struct {
	// Message contains the error message for this step
	Message string
	// Location contains the file:line where this error occurred
	Location string
}

// Unwind extracts all the steps from an error chain, producing a slice
// of Steps that can be used for detailed error reporting and debugging.
// It preserves both the error messages and the locations where they occurred.
//
// Example:
//
//	err := New("base error")
//	err = fault.Wrap(err, fault.With("missing paramter", "The request is missing a parameter"))
//	steps := fault.Unwind(err)
//	for _, step := range steps {
//	    fmt.Printf("Error at %s: %s\n", step.Location, step.Message)
//	}
func Unwind(err error) []Step {
	if err == nil {
		return []Step{}
	}

	chain := []error{}

	for err != nil {
		chain = append(chain, err)
		err = errors.Unwrap(err)
	}

	lastLocation := ""
	steps := []Step{}
	for i := len(chain) - 1; i >= 0; i-- {
		err := chain[i]
		var next error
		if i+1 < len(chain) {
			next = chain[i+1]
		}

		switch unwithLocation := err.(type) {
		case *withLocation:
			{
				_, ok := next.(*withLocation)
				if ok && unwithLocation.location != "" {
					steps = append(steps, Step{Message: "", Location: unwithLocation.location})
				}
				lastLocation = unwithLocation.location
			}
		case *root:
			{
				steps = append(steps, Step{Message: unwithLocation.message, Location: unwithLocation.location})
				lastLocation = ""
			}
		default:
			{
				steps = append(steps, Step{Message: err.Error(), Location: lastLocation})
			}
		}
	}
	slices.Reverse(steps)
	return steps
}
