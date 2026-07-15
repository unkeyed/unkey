package controlplane

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

// connectInterceptor creates a streaming-compatible interceptor that attaches
// the bearer token to every outgoing unary and streaming request.
func connectInterceptor(bearer string) connect.Interceptor {
	return &authInterceptor{bearer: bearer}
}

// authInterceptor implements connect.Interceptor for both unary and streaming calls.
//
// It is stateless and safe for concurrent use.
type authInterceptor struct {
	bearer string
}

// WrapUnary attaches the Authorization header before forwarding the request.
func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		i.setHeaders(req.Header())
		return next(ctx, req)
	}
}

// WrapStreamingClient wraps streaming client calls so that the Authorization
// header is attached on the first Send(). Streaming connections don't expose
// headers until the connection is established, so we can't set them upfront.
func (i *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		return &streamingClientInterceptor{
			StreamingClientConn: next(ctx, spec),
			interceptor:         i,
			spec:                spec,
			headersSet:          false,
		}
	}
}

// WrapStreamingHandler is a passthrough for client-side interceptors; it exists
// only to satisfy [connect.Interceptor].
func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		return next(ctx, conn)
	}
}

// streamingClientInterceptor wraps streaming client connections to attach the
// Authorization header on the first message send.
type streamingClientInterceptor struct {
	connect.StreamingClientConn
	interceptor *authInterceptor
	spec        connect.Spec
	headersSet  bool
}

// Send attaches the Authorization header on the first send, then forwards.
func (s *streamingClientInterceptor) Send(msg any) error {
	if !s.headersSet {
		s.interceptor.setHeaders(s.RequestHeader())
		s.headersSet = true
	}
	return s.StreamingClientConn.Send(msg)
}

// RequestHeader returns the underlying request headers.
func (s *streamingClientInterceptor) RequestHeader() http.Header {
	return s.StreamingClientConn.RequestHeader()
}

// setHeaders sets the Authorization header on the request.
func (i *authInterceptor) setHeaders(header http.Header) {
	header.Set("Authorization", fmt.Sprintf("Bearer %s", i.bearer))
}
