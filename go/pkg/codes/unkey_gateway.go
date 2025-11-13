package codes

// gatewayProxy defines errors related to gateway proxy functionality.
type gatewayProxy struct {
	// BadGateway represents a 502 error - invalid response from upstream server
	BadGateway Code

	// ServiceUnavailable represents a 503 error - backend service is unavailable
	ServiceUnavailable Code

	// GatewayTimeout represents a 504 error - upstream server timeout
	GatewayTimeout Code

	// ProxyForwardFailed represents a 502 error - failed to forward request to backend
	ProxyForwardFailed Code
}

// gatewayRouting defines errors related to gateway routing functionality.
type gatewayRouting struct {
	// ConfigNotFound represents a 404 error - no gateway configuration found for the requested host
	ConfigNotFound Code

	// VMSelectionFailed represents a 500 error - failed to select an available VM
	VMSelectionFailed Code

	// DeploymentDisabled represents a 503 error - deployment exists but is currently disabled
	DeploymentDisabled Code
}

// gatewayAuth defines errors related to gateway authentication functionality.
type gatewayAuth struct {
	// Unauthorized represents a 401 error - authentication required or failed
	Unauthorized Code

	// RateLimited represents a 429 error - rate limit exceeded
	RateLimited Code
}

// gatewayValidation defines errors related to gateway validation functionality.
type gatewayValidation struct {
	// RequestInvalid represents a 400 error - request validation failed
	RequestInvalid Code

	// ResponseInvalid represents a 502 error - response validation failed
	ResponseInvalid Code
}

// gatewayInternal defines errors related to internal gateway functionality.
type gatewayInternal struct {
	// InternalServerError represents a 500 error - internal server error
	InternalServerError Code
	// KeyVerificationFailed represents a 500 error - key verification service failure
	KeyVerificationFailed Code
}

// UnkeyGatewayErrors defines all gateway-related errors in the Unkey system.
// These errors occur when the gateway has issues communicating with backend services.
type UnkeyGatewayErrors struct {
	// Proxy contains errors related to gateway proxy functionality.
	Proxy gatewayProxy

	// Routing contains errors related to gateway routing functionality.
	Routing gatewayRouting

	// Auth contains errors related to gateway authentication functionality.
	Auth gatewayAuth

	// Validation contains errors related to request/response validation functionality.
	Validation gatewayValidation

	// Internal contains errors related to internal gateway functionality.
	Internal gatewayInternal
}

// Gateway contains all predefined gateway error codes.
// These errors can be referenced directly (e.g., codes.Gateway.Proxy.BadGateway)
// for consistent error handling throughout the application.
var Gateway = UnkeyGatewayErrors{
	Proxy: gatewayProxy{
		BadGateway:         Code{SystemUnkey, CategoryBadGateway, "bad_gateway"},
		ServiceUnavailable: Code{SystemUnkey, CategoryServiceUnavailable, "service_unavailable"},
		GatewayTimeout:     Code{SystemUnkey, CategoryGatewayTimeout, "gateway_timeout"},
		ProxyForwardFailed: Code{SystemUnkey, CategoryBadGateway, "proxy_forward_failed"},
	},
	Routing: gatewayRouting{
		ConfigNotFound:     Code{SystemUnkey, CategoryNotFound, "config_not_found"},
		VMSelectionFailed:  Code{SystemUnkey, CategoryInternalServerError, "vm_selection_failed"},
		DeploymentDisabled: Code{SystemUnkey, CategoryServiceUnavailable, "deployment_disabled"},
	},
	Auth: gatewayAuth{
		Unauthorized: Code{SystemUnkey, CategoryUnauthorized, "unauthorized"},
		RateLimited:  Code{SystemUnkey, CategoryRateLimited, "rate_limited"},
	},
	Validation: gatewayValidation{
		RequestInvalid:  Code{SystemUnkey, CategoryUserBadRequest, "request_invalid"},
		ResponseInvalid: Code{SystemUnkey, CategoryUserBadRequest, "response_invalid"},
	},
	Internal: gatewayInternal{
		InternalServerError:   Code{SystemUnkey, CategoryInternalServerError, "internal_server_error"},
		KeyVerificationFailed: Code{SystemUnkey, CategoryInternalServerError, "key_verification_failed"},
	},
}
