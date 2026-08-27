package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadKeyspace authorizes reading keyspace resources.
//
// Valid resource: urn.Keyspace.
type ReadKeyspace struct{}

func (ReadKeyspace) ActionFor(urn.Keyspace) {}
func (ReadKeyspace) String() string         { return "read_keyspace" }

// WriteKeyspace authorizes creating or updating a keyspace.
type WriteKeyspace struct{}

func (WriteKeyspace) ActionFor(urn.Keyspace) {}
func (WriteKeyspace) String() string         { return "write_keyspace" }

// DeleteKeyspace authorizes deleting a keyspace.
type DeleteKeyspace struct{}

func (DeleteKeyspace) ActionFor(urn.Keyspace) {}
func (DeleteKeyspace) String() string         { return "delete_keyspace" }

// CreateKey authorizes creating keys in a keyspace.
//
// Valid resource: urn.Keyspace.
type CreateKey struct{}

func (CreateKey) ActionFor(urn.Keyspace) {}
func (CreateKey) String() string         { return "create_key" }
