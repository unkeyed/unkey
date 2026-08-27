// Package permissions defines typed actions for canonical URN resources.
//
// Each action implements ActionFor for exactly one resource type. The compiler
// rejects invalid resource and action pairs. This package depends on pkg/urn.
// The URN package does not know about actions.
package permissions

import "fmt"

// Action is an action that is valid for resource type R.
type Action[R fmt.Stringer] interface {
	ActionFor(R)
	fmt.Stringer
}

// Wildcard is the action used by the global admin permission.
const Wildcard = "*"
