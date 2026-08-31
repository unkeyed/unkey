package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadRatelimitOverride authorizes reading a rate limit override.
type ReadRatelimitOverride struct{}

func (ReadRatelimitOverride) ActionFor(urn.RatelimitOverride) {}
func (ReadRatelimitOverride) String() string                  { return "read_ratelimit_override" }

// WriteRatelimitOverride authorizes creating or updating a rate limit override.
type WriteRatelimitOverride struct{}

func (WriteRatelimitOverride) ActionFor(urn.RatelimitOverride) {}
func (WriteRatelimitOverride) String() string                  { return "write_ratelimit_override" }

// DeleteRatelimitOverride authorizes deleting a rate limit override.
type DeleteRatelimitOverride struct{}

func (DeleteRatelimitOverride) ActionFor(urn.RatelimitOverride) {}
func (DeleteRatelimitOverride) String() string                  { return "delete_ratelimit_override" }
