package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateKey authorizes creating keys in a keyspace.
//
// Valid resource: urn.Keyspace.
type CreateKey struct{}

func (CreateKey) ActionFor(urn.Keyspace) {}
func (CreateKey) String() string         { return "create_key" }
