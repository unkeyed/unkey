package codes

// gatewayProxy defines errors related to gateway proxy functionality.
type gatewayProxy struct {
	// BadGateway represents a 502 error - invalid response from upstream server
	BadGateway Code

	// ServiceUnavailable represents a 503 error - backend service is unavailable
	ServiceUnavailable Code

	// GatewayTimeout represents a 504 error - upstream server timeout
	GatewayTimeout Code
}

// gatewayRouting defines errors related to gateway routing functionality.
type gatewayRouting struct {
	// ConfigNotFound represents a 404 error - no gateway configuration found for the requested host
	ConfigNotFound Code
}

// UnkeyGatewayErrors defines all gateway-related errors in the Unkey system.
// These errors occur when the gateway has issues communicating with backend services.
type UnkeyGatewayErrors struct {
	// Proxy contains errors related to gateway proxy functionality.
	Proxy gatewayProxy

	// Routing contains errors related to gateway routing functionality.
	Routing gatewayRouting
}

// Gateway contains all predefined gateway error codes.
// These errors can be referenced directly (e.g., codes.Gateway.Proxy.BadGateway)
// for consistent error handling throughout the application.
var Gateway = UnkeyGatewayErrors{
	Proxy: gatewayProxy{
		BadGateway:         Code{SystemUnkey, CategoryBadGateway, "bad_gateway"},
		ServiceUnavailable: Code{SystemUnkey, CategoryBadGateway, "service_unavailable"},
		GatewayTimeout:     Code{SystemUnkey, CategoryBadGateway, "gateway_timeout"},
	},
	Routing: gatewayRouting{
		ConfigNotFound: Code{SystemUnkey, CategoryNotFound, "config_not_found"},
	},
}
