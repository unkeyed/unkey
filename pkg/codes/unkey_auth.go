package codes

// authAuthentication defines errors related to authentication failures.
type authAuthentication struct {
	// Missing indicates authentication credentials were not provided.
	Missing Code

	// Malformed indicates authentication credentials were incorrectly formatted.
	Malformed Code

	// KeyNotFound indicates the authentication key was not found.
	KeyNotFound Code
}

// authAuthorization defines errors related to authorization failures.
type authAuthorization struct {
	// InsufficientPermissions indicates the authenticated entity lacks
	// sufficient permissions for the requested operation.
	InsufficientPermissions Code

	// Forbidden indicates the operation is not allowed.
	Forbidden Code

	// KeyDisabled indicates the authentication key is disabled.
	KeyDisabled Code

	// WorkspaceDisabled indicates the associated workspace is disabled.
	WorkspaceDisabled Code
}

// UnkeyAuthErrors defines all authentication and authorization related errors
// in the Unkey system.
type UnkeyAuthErrors struct {
	// Authentication contains errors related to authentication failures.
	Authentication authAuthentication

	// Authorization contains errors related to authorization failures.
	Authorization authAuthorization
}

// Auth contains all predefined authentication and authorization error codes.
// These errors can be referenced directly (e.g., codes.Auth.Authentication.Missing)
// for consistent error handling throughout the application.
var Auth = UnkeyAuthErrors{
	Authentication: authAuthentication{
		Missing:     Code{SystemUnkey, CategoryUnkeyAuthentication, "missing"},
		Malformed:   Code{SystemUnkey, CategoryUnkeyAuthentication, "malformed"},
		KeyNotFound: Code{SystemUnkey, CategoryUnkeyAuthentication, "key_not_found"},
	},

	Authorization: authAuthorization{
		InsufficientPermissions: Code{SystemUnkey, CategoryUnkeyAuthorization, "insufficient_permissions"},
		Forbidden:               Code{SystemUnkey, CategoryUnkeyAuthorization, "forbidden"},
		KeyDisabled:             Code{SystemUnkey, CategoryUnkeyAuthorization, "key_disabled"},
		WorkspaceDisabled:       Code{SystemUnkey, CategoryUnkeyAuthorization, "workspace_disabled"},
	},
}
