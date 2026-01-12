package interceptor

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
)

// NewHeaderInjector creates an interceptor for cross-cutting concerns like authentication,
// tracing, or API versioning that need to be included in every RPC request without
// coupling each call site to these requirements.
func NewHeaderInjector(headers map[string]string) connect.Interceptor {
	return &headerInterceptor{headers: headers}
}

// headerInterceptor ensures consistent header propagation across all RPC communication
// patterns, preventing inconsistencies that could cause auth failures or lost trace context.
type headerInterceptor struct {
	headers map[string]string
}

// WrapUnary handles request-response RPCs where headers must be set before the single
// request is sent, unlike streaming where we need to defer until the connection is established.
func (i *headerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		i.setHeaders(req.Header())
		return next(ctx, req)
	}
}

// WrapStreamingClient wraps the connection rather than setting headers immediately because
// streaming connections are established first and headers can only be sent with the initial message.
func (i *headerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		return &streamingConn{
			StreamingClientConn: conn,
			interceptor:         i,
		}
	}
}

// WrapStreamingHandler is a no-op because this interceptor only modifies outgoing client
// requests, not incoming server requests where headers are already present.
func (i *headerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// setHeaders centralizes header mutation to ensure consistent behavior across unary
// and streaming calls, avoiding subtle differences that could cause production issues.
func (i *headerInterceptor) setHeaders(header http.Header) {
	for key, value := range i.headers {
		header.Set(key, value)
	}
}

// streamingConn defers header injection until the first Send because streaming protocols
// require the connection to be established before headers can be transmitted.
type streamingConn struct {
	connect.StreamingClientConn
	interceptor *headerInterceptor
	headersSet  bool
}

// Send lazily injects headers on the first message because streaming RPC headers are sent
// with the initial frame, not at connection time, and we only want to set them once.
func (s *streamingConn) Send(msg any) error {
	if !s.headersSet {
		s.interceptor.setHeaders(s.RequestHeader())
		s.headersSet = true
	}
	return s.StreamingClientConn.Send(msg)
}
