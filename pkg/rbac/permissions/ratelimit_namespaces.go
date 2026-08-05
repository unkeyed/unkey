package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateNamespace authorizes creating a rate limit namespace resource.
type CreateNamespace struct{}

func (CreateNamespace) ActionFor(urn.RatelimitNamespace) {}
func (CreateNamespace) String() string                   { return "create_namespace" }

// Limit authorizes consuming rate limits in a namespace.
type Limit struct{}

func (Limit) ActionFor(urn.RatelimitNamespace) {}
func (Limit) String() string                   { return "limit" }

// ReadRatelimitLogs authorizes reading logs for a rate limit namespace.
type ReadRatelimitLogs struct{}

func (ReadRatelimitLogs) ActionFor(urn.RatelimitNamespace) {}
func (ReadRatelimitLogs) String() string                   { return "read_logs" }

// SetOverride authorizes setting a rate limit override resource.
//
// Valid resource: urn.RatelimitOverride.
type SetOverride struct{}

func (SetOverride) ActionFor(urn.RatelimitOverride) {}
func (SetOverride) String() string                  { return "set_override" }
