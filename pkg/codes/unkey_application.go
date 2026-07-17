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

type appPrecondition struct {
	// PreconditionFailed indicates a precondition check failed.
	PreconditionFailed Code

	// DeploymentNotReady indicates the target deployment is not usable: either it
	// never reached ready status or it is shutting down and cannot serve traffic.
	DeploymentNotReady Code

	// NotProductionDeployment indicates the action is only allowed on production
	// deployments.
	NotProductionDeployment Code

	// NoLiveDeployment indicates the app has no live deployment to act over.
	NoLiveDeployment Code

	// DeploymentAlreadyLive indicates the target deployment is already the live
	// deployment.
	DeploymentAlreadyLive Code

	// DeploymentNotRunning indicates the target deployment is not running.
	DeploymentNotRunning Code

	// DeploymentAlreadyStopping indicates a stop is already in flight for the
	// target deployment.
	DeploymentAlreadyStopping Code

	// ProductionCannotStop indicates production deployments cannot be stopped.
	ProductionCannotStop Code

	// DeploymentNotStopped indicates the target deployment is not stopped.
	DeploymentNotStopped Code

	// ProductionCannotStart indicates production deployments cannot be started.
	ProductionCannotStart Code
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

	// Precondition contains errors related to resource preconditions.
	Precondition appPrecondition
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

	Precondition: appPrecondition{
		PreconditionFailed:        Code{SystemUnkey, CategoryUnkeyApplication, "precondition_failed"},
		DeploymentNotReady:        Code{SystemUnkey, CategoryUnkeyApplication, "deployment_not_ready"},
		NotProductionDeployment:   Code{SystemUnkey, CategoryUnkeyApplication, "not_production_deployment"},
		NoLiveDeployment:          Code{SystemUnkey, CategoryUnkeyApplication, "no_live_deployment"},
		DeploymentAlreadyLive:     Code{SystemUnkey, CategoryUnkeyApplication, "deployment_already_live"},
		DeploymentNotRunning:      Code{SystemUnkey, CategoryUnkeyApplication, "deployment_not_running"},
		DeploymentAlreadyStopping: Code{SystemUnkey, CategoryUnkeyApplication, "deployment_already_stopping"},
		ProductionCannotStop:      Code{SystemUnkey, CategoryUnkeyApplication, "production_cannot_stop"},
		DeploymentNotStopped:      Code{SystemUnkey, CategoryUnkeyApplication, "deployment_not_stopped"},
		ProductionCannotStart:     Code{SystemUnkey, CategoryUnkeyApplication, "production_cannot_start"},
	},
}
