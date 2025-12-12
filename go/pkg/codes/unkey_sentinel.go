package codes

// sentinelRouting defines errors related to sentinel routing functionality.
type sentinelRouting struct {
	// DeploymentNotFound represents a 404 error - deployment not found or belongs to wrong environment
	DeploymentNotFound Code

	// NoRunningInstances represents a 503 error - no running instances available for deployment
	NoRunningInstances Code

	// InstanceSelectionFailed represents a 500 error - failed to select an available instance
	InstanceSelectionFailed Code
}

// sentinelProxy defines errors related to sentinel proxy functionality.
type sentinelProxy struct {
	// BadGateway represents a 502 error - invalid response from instance
	BadGateway Code

	// ServiceUnavailable represents a 503 error - instance is unavailable
	ServiceUnavailable Code

	// SentinelTimeout represents a 504 error - instance timeout
	SentinelTimeout Code

	// ProxyForwardFailed represents a 502 error - failed to forward request to instance
	ProxyForwardFailed Code
}

// sentinelInternal defines errors related to internal sentinel functionality.
type sentinelInternal struct {
	// InternalServerError represents a 500 error - internal server error
	InternalServerError Code

	// InvalidConfiguration represents a 500 error - invalid sentinel configuration
	InvalidConfiguration Code
}

// UnkeySentinelErrors defines all sentinel-related errors in the Unkey system.
// These errors occur when the sentinel service has issues routing requests to instances.
type UnkeySentinelErrors struct {
	// Routing contains errors related to sentinel routing functionality.
	Routing sentinelRouting

	// Proxy contains errors related to sentinel proxy functionality.
	Proxy sentinelProxy

	// Internal contains errors related to internal sentinel functionality.
	Internal sentinelInternal
}

// Sentinel contains all predefined sentinel error codes.
// These errors can be referenced directly (e.g., codes.Sentinel.Routing.DeploymentNotFound)
// for consistent error handling throughout the application.
var Sentinel = UnkeySentinelErrors{
	Routing: sentinelRouting{
		DeploymentNotFound:      Code{SystemUnkey, CategoryNotFound, "deployment_not_found"},
		NoRunningInstances:      Code{SystemUnkey, CategoryServiceUnavailable, "no_running_instances"},
		InstanceSelectionFailed: Code{SystemUnkey, CategoryInternalServerError, "instance_selection_failed"},
	},
	Proxy: sentinelProxy{
		BadGateway:         Code{SystemUnkey, CategoryBadGateway, "bad_sentinel"},
		ServiceUnavailable: Code{SystemUnkey, CategoryServiceUnavailable, "service_unavailable"},
		SentinelTimeout:    Code{SystemUnkey, CategoryGatewayTimeout, "sentinel_timeout"},
		ProxyForwardFailed: Code{SystemUnkey, CategoryBadGateway, "proxy_forward_failed"},
	},
	Internal: sentinelInternal{
		InternalServerError:  Code{SystemUnkey, CategoryInternalServerError, "internal_server_error"},
		InvalidConfiguration: Code{SystemUnkey, CategoryInternalServerError, "invalid_configuration"},
	},
}
