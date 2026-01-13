package interceptor

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
)

// NewHeaderInjector creates a streaming-compatible interceptor that adds custom
// headers to all requests (both unary and streaming).
//
// The interceptor automatically injects all provided headers into outgoing requests,
// making it suitable for cross-cutting concerns like:
// - Authentication tokens (Authorization headers)
// - Request routing metadata (region, cluster IDs)
// - Tracing and correlation IDs
// - API versioning headers
//
// This is the recommended way to add headers consistently across all RPC calls
// without coupling each call site to these requirements.
func NewHeaderInjector(headers map[string]string) connect.Interceptor {
	return &headerInterceptor{headers: headers}
}

// headerInterceptor implements connect.Interceptor for both unary and streaming calls.
//
// It automatically adds custom headers to all outgoing requests, ensuring consistent
// header propagation across all RPC communication patterns. This prevents inconsistencies
// that could cause authentication failures, lost trace context, or routing issues.
//
// The interceptor is stateless and safe for concurrent use.
type headerInterceptor struct {
	headers map[string]string
}

// WrapUnary intercepts unary RPC calls by adding required headers before forwarding
// the request to the next handler.
//
// This method adds all configured headers to unary requests. The headers are set
// before the request is processed by the actual service.
func (i *headerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		i.setHeaders(req.Header())
		return next(ctx, req)
	}
}

// WrapStreamingClient intercepts streaming client calls by wrapping the connection
// to add headers on the first message send.
//
// Headers must be set on the first Send() call because streaming connections
// don't expose headers until the connection is established. The wrapper tracks
// whether headers have been set to avoid duplicate header injection.
func (i *headerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return &streamingClientInterceptor{
			StreamingClientConn: next(ctx, spec),
			interceptor:         i,
			spec:                spec,
			headersSet:          false,
		}
	}
}

// WrapStreamingHandler intercepts streaming server calls.
//
// For client-side interceptors, this method is typically not invoked but
// is implemented to satisfy the [connect.Interceptor] interface.
// The implementation simply forwards the call without modification.
func (i *headerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// For client-side interceptors, this is typically not used
		// but we implement it to satisfy the interface
		return next(ctx, conn)
	}
}

// setHeaders adds the configured headers to the provided header map.
//
// This method injects all headers provided during interceptor creation.
// All headers use Set() to overwrite any existing values, ensuring
// consistent behavior across unary and streaming calls.
func (i *headerInterceptor) setHeaders(header http.Header) {
	for key, value := range i.headers {
		header.Set(key, value)
	}
}

// streamingClientInterceptor wraps streaming client connections to add headers
// on the first message send.
//
// This type is needed because streaming connections don't allow header modification
// after establishment. Headers must be injected with the first message payload.
type streamingClientInterceptor struct {
	connect.StreamingClientConn
	interceptor *headerInterceptor
	spec        connect.Spec
	headersSet  bool
}

// Send intercepts outgoing messages in streaming calls by adding configured
// headers on the first message.
//
// Headers are only set once per stream to avoid duplication. Subsequent calls
// forward directly to the underlying connection without modification.
func (s *streamingClientInterceptor) Send(msg any) error {
	// Set headers on the first send
	if !s.headersSet {
		s.interceptor.setHeaders(s.RequestHeader())
		s.headersSet = true
	}
	return s.StreamingClientConn.Send(msg)
}

// RequestHeader returns the request headers for the streaming call.
//
// This method delegates to the underlying streaming connection and is used
// by the Send() method to inject configured headers.
func (s *streamingClientInterceptor) RequestHeader() http.Header {
	return s.StreamingClientConn.RequestHeader()
}
