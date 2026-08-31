package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadOverride authorizes reading rate limit override resources.
//
// Valid resource: urn.RatelimitOverride.
type ReadOverride struct{}

func (ReadOverride) ActionFor(urn.RatelimitOverride) {}
func (ReadOverride) String() string                  { return "read_override" }

// DeleteOverride authorizes deleting rate limit override resources.
//
// Valid resource: urn.RatelimitOverride.
type DeleteOverride struct{}

func (DeleteOverride) ActionFor(urn.RatelimitOverride) {}
func (DeleteOverride) String() string                  { return "delete_override" }
