package proxy

import "strings"

// Header constants for frontline debugging and tracing
const (
	// unkeyHeaderPrefix identifies headers reserved for Frontline.
	unkeyHeaderPrefix = "X-Unkey-"

	// These headers identify the Frontline that handled the request. Frontline
	// sets them on responses and requests to deployment instances.
	HeaderFrontlineID = "X-Unkey-Frontline-Id" // ID of the frontline instance
	HeaderRegion      = "X-Unkey-Region"       // Region of the frontline instance
	HeaderRequestID   = "X-Unkey-Request-Id"   // Request ID for tracing

	// HeaderFrontlineMeta carries signed metadata between Frontline regions.
	HeaderFrontlineMeta = "X-Unkey-Frontline-Meta"
)

// IsUnkeyHeader reports whether a canonical HTTP header name is reserved for
// Frontline.
func IsUnkeyHeader(name string) bool {
	return strings.HasPrefix(name, unkeyHeaderPrefix)
}
