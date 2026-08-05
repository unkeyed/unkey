package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateKeyspace authorizes creating a keyspace resource.
type CreateKeyspace struct{}

func (CreateKeyspace) ActionFor(urn.Keyspace) {}
func (CreateKeyspace) String() string         { return "create_keyspace" }

// ReadKeyspace authorizes reading keyspace resources.
//
// Valid resource: urn.Keyspace.
type ReadKeyspace struct{}

func (ReadKeyspace) ActionFor(urn.Keyspace) {}
func (ReadKeyspace) String() string         { return "read_keyspace" }

// DeleteKeyspace authorizes deleting keyspace resources.
type DeleteKeyspace struct{}

func (DeleteKeyspace) ActionFor(urn.Keyspace) {}
func (DeleteKeyspace) String() string         { return "delete_keyspace" }

// ReadKeyspaceLogs authorizes reading verification logs for a keyspace.
type ReadKeyspaceLogs struct{}

func (ReadKeyspaceLogs) ActionFor(urn.Keyspace) {}
func (ReadKeyspaceLogs) String() string         { return "read_logs" }

// CreateKey authorizes creating a key resource.
//
// Valid resource: urn.Key.
type CreateKey struct{}

func (CreateKey) ActionFor(urn.Key) {}
func (CreateKey) String() string    { return "create_key" }

// CreateKeyInKeyspace preserves the parent-targeted create check used by
// routes that have not migrated to prospective key resources yet.
// Deprecated: use CreateKey with a concrete or wildcard urn.Key.
type CreateKeyInKeyspace struct{}

func (CreateKeyInKeyspace) ActionFor(urn.Keyspace) {}
func (CreateKeyInKeyspace) String() string         { return "create_key" }
