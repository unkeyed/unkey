package codes

// frontlineProxy defines errors related to frontline proxy functionality.
type frontlineProxy struct {
	// BadGateway represents a 502 error - invalid response from upstream server
	BadGateway Code

	// ServiceUnavailable represents a 503 error - backend service is unavailable
	ServiceUnavailable Code

	// GatewayTimeout represents a 504 error - upstream server timeout
	GatewayTimeout Code

	// ProxyForwardFailed represents a 502 error - failed to forward request to backend
	ProxyForwardFailed Code
}

// frontlineRouting defines errors related to frontline routing functionality.
type frontlineRouting struct {
	// ConfigNotFound represents a 404 error - no configuration found for the requested hostname
	ConfigNotFound Code

	// DeploymentSelectionFailed represents a 500 error - failed to select an available deployment
	DeploymentSelectionFailed Code

	// DeploymentDisabled represents a 503 error - all deployments are currently disabled
	DeploymentDisabled Code

	// NoRunningInstances represents a 503 error - no deployments have running instances
	NoRunningInstances Code
}

// frontlineInternal defines errors related to internal frontline functionality.
type frontlineInternal struct {
	// InternalServerError represents a 500 error - internal server error
	InternalServerError Code

	// ConfigLoadFailed represents a 500 error - failed to load configuration
	ConfigLoadFailed Code

	// InstanceLoadFailed represents a 500 error - failed to load instance information
	InstanceLoadFailed Code
}

// UnkeyFrontlineErrors defines all frontline-related errors in the Unkey system.
// These errors occur when the frontline service has issues routing requests to deployments.
type UnkeyFrontlineErrors struct {
	// Proxy contains errors related to frontline proxy functionality.
	Proxy frontlineProxy

	// Routing contains errors related to frontline routing functionality.
	Routing frontlineRouting

	// Internal contains errors related to internal frontline functionality.
	Internal frontlineInternal
}

// Frontline contains all predefined frontline error codes.
// These errors can be referenced directly (e.g., codes.Frontline.Routing.ConfigNotFound)
// for consistent error handling throughout the application.
var Frontline = UnkeyFrontlineErrors{
	Proxy: frontlineProxy{
		BadGateway:         Code{SystemUnkey, CategoryBadGateway, "bad_gateway"},
		ServiceUnavailable: Code{SystemUnkey, CategoryServiceUnavailable, "service_unavailable"},
		GatewayTimeout:     Code{SystemUnkey, CategoryGatewayTimeout, "gateway_timeout"},
		ProxyForwardFailed: Code{SystemUnkey, CategoryBadGateway, "proxy_forward_failed"},
	},
	Routing: frontlineRouting{
		ConfigNotFound:            Code{SystemUnkey, CategoryNotFound, "config_not_found"},
		DeploymentSelectionFailed: Code{SystemUnkey, CategoryInternalServerError, "deployment_selection_failed"},
		DeploymentDisabled:        Code{SystemUnkey, CategoryServiceUnavailable, "deployment_disabled"},
		NoRunningInstances:        Code{SystemUnkey, CategoryServiceUnavailable, "no_running_instances"},
	},
	Internal: frontlineInternal{
		InternalServerError: Code{SystemUnkey, CategoryInternalServerError, "internal_server_error"},
		ConfigLoadFailed:    Code{SystemUnkey, CategoryInternalServerError, "config_load_failed"},
		InstanceLoadFailed:  Code{SystemUnkey, CategoryInternalServerError, "instance_load_failed"},
	},
}
