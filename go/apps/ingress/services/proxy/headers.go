package proxy

// Header constants for ingress debugging and tracing
const (
	// Headers set on BOTH response (to client) AND request (to downstream service)
	// These identify which ingress processed the request
	HeaderIngressID   = "X-Unkey-Ingress-Id"      // ID of the ingress instance
	HeaderRegion      = "X-Unkey-Region"          // Region of the ingress instance
	HeaderRequestID   = "X-Unkey-Request-Id"      // Request ID for tracing
	HeaderIngressTime = "X-Unkey-Ingress-Time-Ms" // Time spent in this ingress (ms)

	// Headers set ONLY on requests to downstream services (gateway/remote ingress)
	// These provide additional context about the forwarding chain
	HeaderParentIngressID = "X-Unkey-Parent-Ingress-Id" // Ingress that forwarded this request
	HeaderParentRequestID = "X-Unkey-Parent-Request-Id" // Original request ID from parent ingress
	HeaderIngressHops     = "X-Unkey-Ingress-Hops"      // Number of ingress hops (loop prevention)
	HeaderDeploymentID    = "X-Deployment-Id"           // Deployment ID for local gateway
	HeaderForwardedProto  = "X-Forwarded-Proto"         // Original protocol (https)
)
