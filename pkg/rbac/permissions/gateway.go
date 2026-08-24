package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadPolicies authorizes reading an environment gateway's policies.
//
// Valid resource: urn.Gateway.
type ReadPolicies struct{}

func (ReadPolicies) ActionFor(urn.Gateway) {}
func (ReadPolicies) String() string        { return "read_policies" }

// WritePolicies authorizes replacing an environment gateway's entire policy
// list or updating a single policy in place.
//
// Valid resource: urn.Gateway.
type WritePolicies struct{}

func (WritePolicies) ActionFor(urn.Gateway) {}
func (WritePolicies) String() string        { return "write_policies" }
