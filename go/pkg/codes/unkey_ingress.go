package codes

// ingressProxy defines errors related to ingress proxy functionality.
type ingressProxy struct {
	// BadGateway represents a 502 error - invalid response from upstream server
	BadGateway Code

	// ServiceUnavailable represents a 503 error - backend service is unavailable
	ServiceUnavailable Code

	// GatewayTimeout represents a 504 error - upstream server timeout
	GatewayTimeout Code

	// ProxyForwardFailed represents a 502 error - failed to forward request to backend
	ProxyForwardFailed Code
}

// ingressRouting defines errors related to ingress routing functionality.
type ingressRouting struct {
	// ConfigNotFound represents a 404 error - no configuration found for the requested hostname
	ConfigNotFound Code

	// DeploymentSelectionFailed represents a 500 error - failed to select an available deployment
	DeploymentSelectionFailed Code

	// DeploymentDisabled represents a 503 error - all deployments are currently disabled
	DeploymentDisabled Code

	// NoRunningInstances represents a 503 error - no deployments have running instances
	NoRunningInstances Code
}

// ingressInternal defines errors related to internal ingress functionality.
type ingressInternal struct {
	// InternalServerError represents a 500 error - internal server error
	InternalServerError Code

	// ConfigLoadFailed represents a 500 error - failed to load configuration
	ConfigLoadFailed Code

	// InstanceLoadFailed represents a 500 error - failed to load instance information
	InstanceLoadFailed Code
}

// UnkeyIngressErrors defines all ingress-related errors in the Unkey system.
// These errors occur when the ingress service has issues routing requests to deployments.
type UnkeyIngressErrors struct {
	// Proxy contains errors related to ingress proxy functionality.
	Proxy ingressProxy

	// Routing contains errors related to ingress routing functionality.
	Routing ingressRouting

	// Internal contains errors related to internal ingress functionality.
	Internal ingressInternal
}

// Ingress contains all predefined ingress error codes.
// These errors can be referenced directly (e.g., codes.Ingress.Routing.ConfigNotFound)
// for consistent error handling throughout the application.
var Ingress = UnkeyIngressErrors{
	Proxy: ingressProxy{
		BadGateway:         Code{SystemUnkey, CategoryBadGateway, "bad_gateway"},
		ServiceUnavailable: Code{SystemUnkey, CategoryServiceUnavailable, "service_unavailable"},
		GatewayTimeout:     Code{SystemUnkey, CategoryGatewayTimeout, "gateway_timeout"},
		ProxyForwardFailed: Code{SystemUnkey, CategoryBadGateway, "proxy_forward_failed"},
	},
	Routing: ingressRouting{
		ConfigNotFound:            Code{SystemUnkey, CategoryNotFound, "config_not_found"},
		DeploymentSelectionFailed: Code{SystemUnkey, CategoryInternalServerError, "deployment_selection_failed"},
		DeploymentDisabled:        Code{SystemUnkey, CategoryServiceUnavailable, "deployment_disabled"},
		NoRunningInstances:        Code{SystemUnkey, CategoryServiceUnavailable, "no_running_instances"},
	},
	Internal: ingressInternal{
		InternalServerError: Code{SystemUnkey, CategoryInternalServerError, "internal_server_error"},
		ConfigLoadFailed:    Code{SystemUnkey, CategoryInternalServerError, "config_load_failed"},
		InstanceLoadFailed:  Code{SystemUnkey, CategoryInternalServerError, "instance_load_failed"},
	},
}
