package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadRatelimitLogs authorizes reading rate limit logs.
type ReadRatelimitLogs struct{}

func (ReadRatelimitLogs) ActionFor(urn.RatelimitLogs) {}
func (ReadRatelimitLogs) String() string              { return "read_ratelimit_logs" }
