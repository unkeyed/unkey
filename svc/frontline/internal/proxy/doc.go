// Package proxy provides HTTP/HTTPS proxying for the frontline.
//
// The service forwards requests to a deployment instance in the local region
// or to a peer frontline in another region. It manages a shared HTTP transport
// for connection pooling, writes timing headers for troubleshooting, and
// captures request/response bodies for ClickHouse logging on the local path.
//
// # Header Management
//
// The service writes identifying headers (frontline ID, region, request ID) on
// responses and deployment instance requests. Timing details are recorded with
// the shared X-Unkey-Timing header using the timing schema.
// X-Unkey-Frontline-Meta carries signed forwarding metadata only on peer
// Frontline requests.
//
// # Loop Prevention
//
// The service tracks a signed hop history in X-Unkey-Frontline-Meta and uses
// its length as the hop count. Each entry records the Frontline ID, region,
// request ID, and forward time. When a request reaches MaxHops, the service
// rejects another cross-region forward.
//
// # Connection Pooling
//
// The peer-frontline transport uses conservative pooling and timeout settings.
// A separate per-protocol transport registry handles upstream instance forwards
// (http1 vs h2c).
//
// # Error Handling
//
// Errors raised by the policy engine or routing surface as fault errors that
// the observability middleware translates into JSON or HTML error responses
// based on the client's Accept header. Upstream responses (instance or peer
// frontline) stream through unchanged.
package proxy
