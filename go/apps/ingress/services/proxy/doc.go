// Package proxy provides HTTP/HTTPS proxying services for the ingress.
//
// The proxy service is responsible for:
//   - Forwarding requests to local gateways (HTTP)
//   - Forwarding requests to remote ingresses (HTTPS)
//   - Managing a shared HTTP transport for connection pooling
//   - Setting debug headers for tracing and troubleshooting
//   - Writing clean error responses (JSON/HTML) when proxying fails
//
// # Header Management
//
// The service sets headers in TWO places for different purposes:
//
// 1. Response headers (back to client):
//   - X-Unkey-Ingress-ID: Which ingress handled the request
//   - X-Unkey-Region: Which region the ingress is in
//   - X-Unkey-Request-ID: Request ID for tracing
//
// These help with debugging and support tickets.
//
// 2. Request headers (to downstream service):
//   - Same headers as above, telling the downstream service who forwarded the request
//   - X-Unkey-Ingress-Time-Ms: Latency added by this ingress
//   - X-Unkey-Parent-Ingress-ID: Previous ingress in the chain (remote only)
//   - X-Unkey-Parent-Request-ID: Original request ID from parent (remote only)
//   - X-Unkey-Ingress-Hops: Hop count for loop prevention (remote only)
//
// # Loop Prevention
//
// The service tracks hop count via X-Unkey-Ingress-Hops header and enforces
// a configurable maximum (default: 3). When a request exceeds MaxHops, it's
// rejected to prevent infinite routing loops. A warning is logged when the
// hop count reaches MaxHops-1 to help identify potential routing issues.
//
// # Connection Pooling
//
// The service uses a shared http.Transport with:
//   - 200 max idle connections
//   - 100 max idle connections per host
//   - 90s idle timeout
//   - 10s TLS handshake timeout
//   - 30s response header timeout
//
// This allows efficient connection reuse across all proxied requests,
// reducing latency and overhead.
//
// # Error Handling
//
// When proxying fails, the service writes clean error responses based on
// the client's Accept header:
//   - JSON errors for API clients (Accept: application/json)
//   - HTML errors for browsers (Accept: text/html)
//
// Errors include the ingress ID and request ID for debugging.
package proxy
