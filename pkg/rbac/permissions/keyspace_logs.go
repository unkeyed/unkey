package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadKeyspaceLogs authorizes reading keyspace logs.
type ReadKeyspaceLogs struct{}

func (ReadKeyspaceLogs) ActionFor(urn.KeyspaceLogs) {}
func (ReadKeyspaceLogs) String() string             { return "read_keyspace_logs" }
