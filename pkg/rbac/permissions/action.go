// Package permissions defines typed actions for Unkey resource permissions.
//
// Resource permissions combine a URN resource with an action, but not every
// action is meaningful for every resource. These action structs encode valid
// resource/action pairs in the type system, so handler code can't accidentally
// ask for create_key on a key resource or read_key on a keyspace resource.
//
// Each action declares the resource type it supports through ActionFor. For
// example, CreateKey implements ActionFor(urn.Keyspace), so rbac.U accepts it
// with urn.Keyspace values and rejects it with urn.Key values at compile time.
//
// This keeps pkg/urn focused on naming resources while pkg/rbac owns query
// construction and evaluation.
package permissions

import "fmt"

// Action is an action that is valid for resource type R.
type Action[R fmt.Stringer] interface {
	ActionFor(R)
	fmt.Stringer
}
