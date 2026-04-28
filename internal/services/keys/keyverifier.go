package keys

import (
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
)

// AuthorizedWorkspaceID returns the workspace ID this key is scoped to act on.
// For root keys this is ForWorkspaceID; for workspace-internal keys this is
// the key's own workspace.
func (k *KeyVerifier) AuthorizedWorkspaceID() string {
	return k.authorizedWorkspaceID
}

// Permissions returns the flat list of granted permission strings.
func (k *KeyVerifier) Permissions() []string {
	return k.permissions
}

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
		permissions:           args.Permissions,
		authorizedWorkspaceID: args.AuthorizedWorkspaceID,
		Status:                args.Status,
	}
}
