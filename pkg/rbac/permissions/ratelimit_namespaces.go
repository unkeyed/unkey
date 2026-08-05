package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateNamespace authorizes creating a rate limit namespace resource.
type CreateNamespace struct{}

func (CreateNamespace) ActionFor(urn.RatelimitNamespace) {}
func (CreateNamespace) String() string                   { return "create_namespace" }

// CreateOverride authorizes creating or replacing a rate limit override.
//
// Valid resource: urn.RatelimitOverride.
type CreateOverride struct{}

func (CreateOverride) ActionFor(urn.RatelimitOverride) {}
func (CreateOverride) String() string                  { return "create_override" }

// Limit authorizes consuming rate limits in a namespace.
type Limit struct{}

func (Limit) ActionFor(urn.RatelimitNamespace) {}
func (Limit) String() string                   { return "limit" }

// ReadRatelimitLogs authorizes reading logs for a rate limit namespace.
type ReadRatelimitLogs struct{}

func (ReadRatelimitLogs) ActionFor(urn.RatelimitNamespace) {}
func (ReadRatelimitLogs) String() string                   { return "read_logs" }
