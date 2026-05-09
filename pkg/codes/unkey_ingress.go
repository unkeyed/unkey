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

	// DeploymentNotFound represents a 404 error - the resolved deployment was
	// not found or did not match the expected environment.
	DeploymentNotFound Code

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

	// InvalidConfiguration represents a 500 error - the deployment's policy
	// configuration could not be parsed.
	InvalidConfiguration Code
}

// frontlineAuth defines errors raised by the policy engine while authenticating
// the caller (KeyAuth, JWTAuth, ...).
type frontlineAuth struct {
	// MissingCredentials represents a 401 error - no credentials found in the request.
	MissingCredentials Code

	// InvalidKey represents a 401 error - key not found, disabled, or expired.
	InvalidKey Code

	// InsufficientPermissions represents a 403 error - the credential lacks
	// the permissions required by a permission_query.
	InsufficientPermissions Code

	// RateLimited represents a 429 error - the credential or its auto-applied
	// rate limit was exceeded.
	RateLimited Code
}

// frontlineFirewall defines errors raised by the Firewall policy.
type frontlineFirewall struct {
	// Denied represents a 403 error - request rejected by a Firewall policy
	// with action=DENY.
	Denied Code
}

// frontlineOpenApi defines errors raised by the OpenAPI request validation policy.
type frontlineOpenApi struct {
	// InvalidRequest represents a 400 error - request does not conform to the OpenAPI spec
	InvalidRequest Code
}

// UnkeyFrontlineErrors defines all frontline-related errors in the Unkey system.
// These errors occur when the frontline service has issues serving a request —
// from routing through policy evaluation to upstream proxying.
type UnkeyFrontlineErrors struct {
	// Proxy contains errors related to frontline proxy functionality.
	Proxy frontlineProxy

	// Routing contains errors related to frontline routing functionality.
	Routing frontlineRouting

	// Internal contains errors related to internal frontline functionality.
	Internal frontlineInternal

	// Auth contains errors raised by the policy engine while authenticating
	// the caller.
	Auth frontlineAuth

	// Firewall contains errors raised by the Firewall policy.
	Firewall frontlineFirewall

	// OpenApi contains errors raised by the OpenAPI request validation policy.
	OpenApi frontlineOpenApi
}

// Frontline contains all predefined frontline error codes.
// These errors can be referenced directly (e.g., codes.Frontline.Routing.ConfigNotFound)
// for consistent error handling throughout the application.
var Frontline = UnkeyFrontlineErrors{
	Proxy: frontlineProxy{
		BadGateway:         Code{SystemUnkey, CategoryBadGateway, "bad_gateway", KindUnknown},
		ServiceUnavailable: Code{SystemUnkey, CategoryServiceUnavailable, "service_unavailable", KindUnknown},
		GatewayTimeout:     Code{SystemUnkey, CategoryGatewayTimeout, "gateway_timeout", KindUnknown},
		ProxyForwardFailed: Code{SystemUnkey, CategoryBadGateway, "proxy_forward_failed", KindUnknown},
	},
	Routing: frontlineRouting{
		ConfigNotFound:            Code{SystemUnkey, CategoryNotFound, "config_not_found", KindNotFound},
		DeploymentNotFound:        Code{SystemUnkey, CategoryNotFound, "deployment_not_found", KindNotFound},
		DeploymentSelectionFailed: Code{SystemUnkey, CategoryInternalServerError, "deployment_selection_failed", KindUnknown},
		DeploymentDisabled:        Code{SystemUnkey, CategoryServiceUnavailable, "deployment_disabled", KindUnknown},
		NoRunningInstances:        Code{SystemUnkey, CategoryServiceUnavailable, "no_running_instances", KindUnknown},
	},
	Internal: frontlineInternal{
		InternalServerError:  Code{SystemUnkey, CategoryInternalServerError, "internal_server_error", KindUnknown},
		ConfigLoadFailed:     Code{SystemUnkey, CategoryInternalServerError, "config_load_failed", KindUnknown},
		InstanceLoadFailed:   Code{SystemUnkey, CategoryInternalServerError, "instance_load_failed", KindUnknown},
		InvalidConfiguration: Code{SystemUnkey, CategoryInternalServerError, "invalid_configuration", KindUnknown},
	},
	Auth: frontlineAuth{
		MissingCredentials:      Code{SystemUnkey, CategoryUnauthorized, "missing_credentials", KindUnknown},
		InvalidKey:              Code{SystemUnkey, CategoryUnauthorized, "invalid_key", KindUnknown},
		InsufficientPermissions: Code{SystemUnkey, CategoryForbidden, "insufficient_permissions", KindUnknown},
		RateLimited:             Code{SystemUnkey, CategoryRateLimited, "rate_limited", KindUnknown},
	},
	Firewall: frontlineFirewall{
		Denied: Code{SystemUnkey, CategoryForbidden, "firewall_denied", KindUnknown},
	},
	OpenApi: frontlineOpenApi{
		InvalidRequest: Code{SystemUnkey, CategoryUserBadRequest, "openapi_validation_failed", KindUnknown},
	},
}
