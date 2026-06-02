package codes

// frontlineProxy defines errors related to frontline proxy functionality.
//
// Each Specific value names one wire-level mechanism the proxy observed
// (ECONNREFUSED, ECONNRESET, EHOSTUNREACH, dial timeout, response
// timeout, our outer deadline). Peer-frontline variants exist so
// failures on the peer hop are distinguishable from failures talking to
// a customer instance.
type frontlineProxy struct {
	// UpstreamConnectionRefused represents a 503 error - ECONNREFUSED dialing a customer instance.
	UpstreamConnectionRefused Code

	// UpstreamHostUnreachable represents a 503 error - EHOSTUNREACH dialing a customer instance.
	UpstreamHostUnreachable Code

	// UpstreamConnectionReset represents a 502 error - ECONNRESET on the upstream connection.
	UpstreamConnectionReset Code

	// DialTimeout represents a 504 error - dial-phase timeout to a customer instance.
	DialTimeout Code

	// UpstreamResponseTimeout represents a 504 error - upstream did not respond in time after dial succeeded.
	UpstreamResponseTimeout Code

	// GatewayDeadlineExceeded represents a 504 error - our own outer request deadline expired.
	GatewayDeadlineExceeded Code

	// ProxyErrorUnclassified represents a 502 error - upstream/dial error
	// the classifier did not recognise. Fallback bucket.
	ProxyErrorUnclassified Code

	// PeerFrontlineConnectionRefused represents a 503 error - ECONNREFUSED to a peer frontline.
	PeerFrontlineConnectionRefused Code

	// PeerFrontlineHostUnreachable represents a 503 error - EHOSTUNREACH to a peer frontline.
	PeerFrontlineHostUnreachable Code

	// PeerFrontlineConnectionReset represents a 502 error - ECONNRESET on a peer-frontline connection.
	PeerFrontlineConnectionReset Code

	// PeerFrontlineDNSNotFound represents a 503 error - DNS NXDOMAIN resolving a peer-frontline hostname.
	PeerFrontlineDNSNotFound Code

	// PeerFrontlineDNSTimeout represents a 504 error - DNS timeout resolving a peer-frontline hostname.
	PeerFrontlineDNSTimeout Code

	// PeerFrontlineTimeout represents a 504 error - timeout reaching a peer frontline.
	PeerFrontlineTimeout Code
}

// frontlineRouting defines errors related to frontline routing functionality.
type frontlineRouting struct {
	// ConfigNotFoundForUnkeyHostname represents a 404 error - request hit
	// a subdomain of the configured default domain and no route is
	// configured for it.
	ConfigNotFoundForUnkeyHostname Code

	// ConfigNotFoundForCustomDomain represents a 404 error - request hit
	// a hostname outside the default domain and no route is configured
	// for it.
	ConfigNotFoundForCustomDomain Code

	// DeploymentNotFound represents a 404 error - the resolved deployment was
	// not found or did not match the expected environment.
	DeploymentNotFound Code

	// DeploymentSelectionFailed represents a 500 error - failed to select an available deployment
	DeploymentSelectionFailed Code

	// NoDeploymentInstances represents a 503 error - the deployment has
	// zero instance rows in the registry.
	NoDeploymentInstances Code

	// NoRunningInstances represents a 503 error - instance rows exist but
	// none are in status=Running.
	NoRunningInstances Code

	// NoReachableRegion represents a 503 error - running instances exist
	// in other regions but the proximity table on this region maps to
	// none of them.
	NoReachableRegion Code
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
// These errors can be referenced directly (e.g., codes.Frontline.Routing.DeploymentNotFound)
// for consistent error handling throughout the application.
var Frontline = UnkeyFrontlineErrors{
	Proxy: frontlineProxy{
		UpstreamConnectionRefused:      Code{SystemUnkey, CategoryServiceUnavailable, "upstream_connection_refused"},
		UpstreamHostUnreachable:        Code{SystemUnkey, CategoryServiceUnavailable, "upstream_host_unreachable"},
		UpstreamConnectionReset:        Code{SystemUnkey, CategoryBadGateway, "upstream_connection_reset"},
		DialTimeout:                    Code{SystemUnkey, CategoryGatewayTimeout, "dial_timeout"},
		UpstreamResponseTimeout:        Code{SystemUnkey, CategoryGatewayTimeout, "upstream_response_timeout"},
		GatewayDeadlineExceeded:        Code{SystemUnkey, CategoryGatewayTimeout, "gateway_deadline_exceeded"},
		ProxyErrorUnclassified:         Code{SystemUnkey, CategoryBadGateway, "proxy_error_unclassified"},
		PeerFrontlineConnectionRefused: Code{SystemUnkey, CategoryServiceUnavailable, "peer_frontline_connection_refused"},
		PeerFrontlineHostUnreachable:   Code{SystemUnkey, CategoryServiceUnavailable, "peer_frontline_host_unreachable"},
		PeerFrontlineConnectionReset:   Code{SystemUnkey, CategoryBadGateway, "peer_frontline_connection_reset"},
		PeerFrontlineDNSNotFound:       Code{SystemUnkey, CategoryServiceUnavailable, "peer_frontline_dns_not_found"},
		PeerFrontlineDNSTimeout:        Code{SystemUnkey, CategoryGatewayTimeout, "peer_frontline_dns_timeout"},
		PeerFrontlineTimeout:           Code{SystemUnkey, CategoryGatewayTimeout, "peer_frontline_timeout"},
	},
	Routing: frontlineRouting{
		ConfigNotFoundForUnkeyHostname: Code{SystemUnkey, CategoryNotFound, "config_not_found_for_unkey_hostname"},
		ConfigNotFoundForCustomDomain:  Code{SystemUnkey, CategoryNotFound, "config_not_found_for_custom_domain"},
		DeploymentNotFound:             Code{SystemUnkey, CategoryNotFound, "deployment_not_found"},
		DeploymentSelectionFailed:      Code{SystemUnkey, CategoryInternalServerError, "deployment_selection_failed"},
		NoDeploymentInstances:          Code{SystemUnkey, CategoryServiceUnavailable, "no_deployment_instances"},
		NoRunningInstances:             Code{SystemUnkey, CategoryServiceUnavailable, "no_running_instances"},
		NoReachableRegion:              Code{SystemUnkey, CategoryServiceUnavailable, "no_reachable_region"},
	},
	Internal: frontlineInternal{
		InternalServerError:  Code{SystemUnkey, CategoryInternalServerError, "internal_server_error"},
		ConfigLoadFailed:     Code{SystemUnkey, CategoryInternalServerError, "config_load_failed"},
		InstanceLoadFailed:   Code{SystemUnkey, CategoryInternalServerError, "instance_load_failed"},
		InvalidConfiguration: Code{SystemUnkey, CategoryInternalServerError, "invalid_configuration"},
	},
	Auth: frontlineAuth{
		MissingCredentials:      Code{SystemUnkey, CategoryUnauthorized, "missing_credentials"},
		InvalidKey:              Code{SystemUnkey, CategoryUnauthorized, "invalid_key"},
		InsufficientPermissions: Code{SystemUnkey, CategoryForbidden, "insufficient_permissions"},
		RateLimited:             Code{SystemUnkey, CategoryRateLimited, "rate_limited"},
	},
	Firewall: frontlineFirewall{
		Denied: Code{SystemUnkey, CategoryForbidden, "firewall_denied"},
	},
	OpenApi: frontlineOpenApi{
		InvalidRequest: Code{SystemUnkey, CategoryUserBadRequest, "openapi_validation_failed"},
	},
}
