package fault

import (
	"slices"
)

type Step struct {
	Message  string
	Location string
}

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
			// if it's not a wrapepd error, then we don't have any more public messages
			// and can stop looking.
			break
		}
		current = next
	}

	slices.Reverse(steps)

	return steps

}
