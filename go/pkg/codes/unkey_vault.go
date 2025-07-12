package codes

// Resource-specific error categories

type vaultPrecondition struct {
	// PreconditionFailed indicates a precondition check failed.
	PreconditionFailed Code
}

// UnkeyVaultErrors defines all vault-related errors in the Unkey system.
// These errors generally relate to the vault's operation rather than
// specific domain entities.
type UnkeyVaultErrors struct {
	// Precondition contains errors related to resource preconditions.
	Precondition vaultPrecondition
}

var Vault = UnkeyVaultErrors{
	Precondition: vaultPrecondition{
		PreconditionFailed: Code{SystemUnkey, CategoryUnkeyVault, "precondition_failed"},
	},
}
