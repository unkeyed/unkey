package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadKeyspace authorizes reading keyspace resources.
//
// Valid resource: urn.Keyspace.
type ReadKeyspace struct{}

func (ReadKeyspace) ActionFor(urn.Keyspace) {}
func (ReadKeyspace) String() string         { return "read_keyspace" }

// CreateKey authorizes creating keys in a keyspace.
//
// Valid resource: urn.Keyspace.
type CreateKey struct{}

func (CreateKey) ActionFor(urn.Keyspace) {}
func (CreateKey) String() string         { return "create_key" }
