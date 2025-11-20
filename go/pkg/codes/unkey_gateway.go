package codes

// gatewayRouting defines errors related to gateway routing functionality.
type gatewayRouting struct {
	// DeploymentNotFound represents a 404 error - deployment not found or belongs to wrong environment
	DeploymentNotFound Code

	// NoRunningInstances represents a 503 error - no running instances available for deployment
	NoRunningInstances Code

	// InstanceSelectionFailed represents a 500 error - failed to select an available instance
	InstanceSelectionFailed Code
}

// gatewayProxy defines errors related to gateway proxy functionality.
type gatewayProxy struct {
	// BadGateway represents a 502 error - invalid response from instance
	BadGateway Code

	// ServiceUnavailable represents a 503 error - instance is unavailable
	ServiceUnavailable Code

	// GatewayTimeout represents a 504 error - instance timeout
	GatewayTimeout Code

	// ProxyForwardFailed represents a 502 error - failed to forward request to instance
	ProxyForwardFailed Code
}

// gatewayInternal defines errors related to internal gateway functionality.
type gatewayInternal struct {
	// InternalServerError represents a 500 error - internal server error
	InternalServerError Code

	// InvalidConfiguration represents a 500 error - invalid gateway configuration
	InvalidConfiguration Code
}

// UnkeyGatewayErrors defines all gateway-related errors in the Unkey system.
// These errors occur when the gateway service has issues routing requests to instances.
type UnkeyGatewayErrors struct {
	// Routing contains errors related to gateway routing functionality.
	Routing gatewayRouting

	// Proxy contains errors related to gateway proxy functionality.
	Proxy gatewayProxy

	// Internal contains errors related to internal gateway functionality.
	Internal gatewayInternal
}

// Gateway contains all predefined gateway error codes.
// These errors can be referenced directly (e.g., codes.Gateway.Routing.DeploymentNotFound)
// for consistent error handling throughout the application.
var Gateway = UnkeyGatewayErrors{
	Routing: gatewayRouting{
		DeploymentNotFound:      Code{SystemUnkey, CategoryNotFound, "deployment_not_found"},
		NoRunningInstances:      Code{SystemUnkey, CategoryServiceUnavailable, "no_running_instances"},
		InstanceSelectionFailed: Code{SystemUnkey, CategoryInternalServerError, "instance_selection_failed"},
	},
	Proxy: gatewayProxy{
		BadGateway:         Code{SystemUnkey, CategoryBadGateway, "bad_gateway"},
		ServiceUnavailable: Code{SystemUnkey, CategoryServiceUnavailable, "service_unavailable"},
		GatewayTimeout:     Code{SystemUnkey, CategoryGatewayTimeout, "gateway_timeout"},
		ProxyForwardFailed: Code{SystemUnkey, CategoryBadGateway, "proxy_forward_failed"},
	},
	Internal: gatewayInternal{
		InternalServerError:  Code{SystemUnkey, CategoryInternalServerError, "internal_server_error"},
		InvalidConfiguration: Code{SystemUnkey, CategoryInternalServerError, "invalid_configuration"},
	},
}
