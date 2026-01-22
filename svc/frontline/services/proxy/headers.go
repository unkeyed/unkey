package proxy

// Header constants for frontline debugging and tracing
const (
	// Headers set on BOTH response (to client) AND request (to downstream service)
	// These identify which frontline processed the request
	HeaderFrontlineID   = "X-Unkey-Frontline-Id"      // ID of the frontline instance
	HeaderRegion        = "X-Unkey-Region"            // Region of the frontline instance
	HeaderRequestID     = "X-Unkey-Request-Id"        // Request ID for tracing
	HeaderFrontlineTime = "X-Unkey-Frontline-Time-Ms" // Time spent in this frontline (ms)

	// Headers set ONLY on requests to downstream services (sentinel/remote frontline)
	// These provide additional context about the forwarding chain
	HeaderParentFrontlineID = "X-Unkey-Parent-Frontline-Id" // Frontline that forwarded this request
	HeaderParentRequestID   = "X-Unkey-Parent-Request-Id"   // Original request ID from parent frontline
	HeaderFrontlineHops     = "X-Unkey-Frontline-Hops"      // Number of frontline hops (loop prevention)
	HeaderDeploymentID      = "X-Deployment-Id"             // Deployment ID for local sentinel
	HeaderForwardedProto    = "X-Forwarded-Proto"           // Original protocol (https)

	// Security headers
	// HSTS tells browsers to only use HTTPS for 1 year, including subdomains
	HeaderHSTS = "Strict-Transport-Security"
	// HSTS value: 1 year max-age with includeSubDomains
	HeaderHSTSValue = "max-age=31536000; includeSubDomains"
)
