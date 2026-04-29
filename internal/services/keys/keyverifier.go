package keys

import (
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
)

// VerifierArgs configures NewVerifier.
type VerifierArgs struct {
	Key                   keysdb.FindKeyForVerificationRow
	Roles                 []string
	Permissions           []string
	AuthorizedWorkspaceID string
	Status                KeyStatus
}

// NewVerifier constructs a KeyVerifier from explicit parts. It exists so tests
// and synthetic call sites outside this package can build verifiers without
// reaching into unexported fields. Production code paths must use
// Service.Get or Service.GetRootKey, which load from the database and attach
// the runtime services this constructor leaves unset.
func NewVerifier(args VerifierArgs) *KeyVerifier {
	// nolint:exhaustruct
	return &KeyVerifier{
		Key:                   args.Key,
		Roles:                 args.Roles,
		Permissions:           args.Permissions,
		AuthorizedWorkspaceID: args.AuthorizedWorkspaceID,
		Status:                args.Status,
	}
}
