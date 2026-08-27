package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadRatelimitNamespace authorizes reading a rate limit namespace.
type ReadRatelimitNamespace struct{}

func (ReadRatelimitNamespace) ActionFor(urn.RatelimitNamespace) {}
func (ReadRatelimitNamespace) String() string                   { return "read_ratelimit_namespace" }

// WriteRatelimitNamespace authorizes creating or updating a rate limit namespace.
type WriteRatelimitNamespace struct{}

func (WriteRatelimitNamespace) ActionFor(urn.RatelimitNamespace) {}
func (WriteRatelimitNamespace) String() string                   { return "write_ratelimit_namespace" }

// DeleteRatelimitNamespace authorizes deleting a rate limit namespace.
type DeleteRatelimitNamespace struct{}

func (DeleteRatelimitNamespace) ActionFor(urn.RatelimitNamespace) {}
func (DeleteRatelimitNamespace) String() string                   { return "delete_ratelimit_namespace" }

// LimitRatelimitNamespace authorizes checking or using a rate limit namespace.
type LimitRatelimitNamespace struct{}

func (LimitRatelimitNamespace) ActionFor(urn.RatelimitNamespace) {}
func (LimitRatelimitNamespace) String() string                   { return "limit_ratelimit_namespace" }
