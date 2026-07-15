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
// both responses and downstream requests. Timing details are recorded with the
// shared X-Unkey-Timing header using the timing schema. Forwarding metadata
// such as parent frontline and hop counts are only attached to peer-frontline
// requests.
//
// # Loop Prevention
//
// The service tracks hop count via X-Unkey-Frontline-Hops and enforces a
// configurable maximum. When a request exceeds MaxHops, it is rejected to
// prevent infinite routing loops between peer frontlines.
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
