// Package proxy provides HTTP/HTTPS proxying services for the frontline.
//
// The proxy service forwards requests to local sentinels or remote frontlines,
// manages a shared HTTP transport for connection pooling, writes timing headers
// for troubleshooting, and returns clean JSON or HTML error responses on failure.
//
// # Header Management
//
// The service writes identifying headers (frontline ID, region, request ID) on
// both responses and downstream requests. Timing details are recorded with the
// shared X-Unkey-Timing header using the timing schema. Forwarding metadata such
// as parent frontline and hop counts are only attached to downstream requests.
//
// # Loop Prevention
//
// The service tracks hop count via X-Unkey-Frontline-Hops and enforces a
// configurable maximum (default: 3). When a request exceeds MaxHops, it is
// rejected to prevent infinite routing loops.
//
// # Connection Pooling
//
// The service uses a shared http.Transport with conservative pooling and
// timeout settings to reduce latency by reusing connections safely.
//
// # Error Handling
//
// When proxying fails, the service writes clean JSON or HTML errors based on the
// client's Accept header and includes frontline identifiers for debugging.
package proxy
