package fault

import (
	"slices"
)

// Step represents a single frame in an error chain, capturing the internal
// message and source location where the error was wrapped. Steps are ordered
// from the root cause to the outermost wrapper when returned by [Flatten].
type Step struct {
	Message  string
	Location string
}

// Flatten unwraps a chain of wrapped errors and returns each frame as a [Step].
// The returned slice is ordered from root cause to outermost wrapper, making it
// suitable for logging or displaying error traces. Returns an empty slice if
// err is nil or not a wrapped error from this package.
func Flatten(err error) []Step {
	if err == nil {
		return []Step{}
	}

	current, ok := err.(*wrapped)
	if !ok {
		return []Step{}
	}

	steps := []Step{}

	for current != nil {
		steps = append(steps, Step{
			Message:  current.internal,
			Location: current.location,
		})

		// Check if there's a next error in the chain
		if current.err == nil {
			break
		}

		// Try to get the next wrapped error
		next, ok := current.err.(*wrapped)
		if !ok {
			// if it's not a wrapped error, then we don't have any more public messages
			// and can stop looking.
			break
		}
		current = next
	}

	slices.Reverse(steps)

	return steps

}
