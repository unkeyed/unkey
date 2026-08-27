package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadKeyspace authorizes reading a keyspace.
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
