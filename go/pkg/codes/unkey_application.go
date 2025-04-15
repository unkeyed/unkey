package codes

// appInternal defines errors related to unexpected internal failures in the
// Unkey application.
type appInternal struct {
	// UnexpectedError represents an unhandled or unexpected error condition.
	UnexpectedError Code

	// ServiceUnavailable indicates a service is temporarily unavailable.
	ServiceUnavailable Code
}

// appValidation defines errors related to input validation failures.
type appValidation struct {
	// InvalidInput indicates a client provided input that failed validation.
	InvalidInput Code

	// AssertionFailed indicates a runtime assertion or invariant check failed.
	AssertionFailed Code
}

// appProtection defines errors related to resource protection mechanisms.
type appProtection struct {
	// ProtectedResource indicates an attempt to modify a protected resource.
	ProtectedResource Code
}

// UnkeyAppErrors defines all application-level errors in the Unkey system.
// These errors generally relate to the application's operation rather than
// specific domain entities.
type UnkeyAppErrors struct {
	// Internal contains errors related to unexpected internal failures.
	Internal appInternal

	// Validation contains errors related to input validation.
	Validation appValidation

	// Protection contains errors related to resource protection.
	Protection appProtection
}

// App contains all predefined application-level error codes.
// These errors can be referenced directly (e.g., codes.App.Internal.UnexpectedError)
// for consistent error handling throughout the application.
var App = UnkeyAppErrors{
	Internal: appInternal{
		UnexpectedError:    Code{SystemUnkey, CategoryUnkeyApplication, "unexpected_error"},
		ServiceUnavailable: Code{SystemUnkey, CategoryUnkeyApplication, "service_unavailable"},
	},

	Validation: appValidation{
		InvalidInput:    Code{SystemUnkey, CategoryUnkeyApplication, "invalid_input"},
		AssertionFailed: Code{SystemUnkey, CategoryUnkeyApplication, "assertion_failed"},
	},

	Protection: appProtection{
		ProtectedResource: Code{SystemUnkey, CategoryUnkeyApplication, "protected_resource"},
	},
}
