package sync

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

// connectInterceptor creates a streaming-compatible interceptor that adds authentication
// and metadata headers to all requests (both unary and streaming)
func connectInterceptor(region, shard, bearer string) connect.Interceptor {
	return &authInterceptor{
		region: region,
		shard:  shard,
		bearer: bearer,
	}
}

// authInterceptor implements connect.Interceptor for both unary and streaming calls
type authInterceptor struct {
	region string
	shard  string
	bearer string
}

// WrapUnary intercepts unary RPC calls
func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		i.setHeaders(req.Header())
		return next(ctx, req)
	}
}

// WrapStreamingClient intercepts streaming client calls
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

// WrapStreamingHandler intercepts streaming server calls (not typically needed for client, but required by interface)
func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		// For client-side interceptors, this is typically not used
		// but we implement it to satisfy the interface
		return next(ctx, conn)
	}
}

// streamingClientInterceptor wraps streaming client connections
type streamingClientInterceptor struct {
	connect.StreamingClientConn
	interceptor *authInterceptor
	spec        connect.Spec
	headersSet  bool
}

// Send intercepts outgoing messages in streaming calls
func (s *streamingClientInterceptor) Send(msg any) error {
	// Set headers on the first send
	if !s.headersSet {
		s.interceptor.setHeaders(s.RequestHeader())
		s.headersSet = true
	}
	return s.StreamingClientConn.Send(msg)
}

// RequestHeader returns the request headers for the streaming call
func (s *streamingClientInterceptor) RequestHeader() http.Header {
	return s.StreamingClientConn.RequestHeader()
}

// setHeaders adds the required authentication and metadata headers
func (i *authInterceptor) setHeaders(header http.Header) {
	header.Set("X-Krane-Region", i.region)
	header.Set("X-Krane-Shard", i.shard)
	header.Set("Authorization", fmt.Sprintf("Bearer %s", i.bearer))
}
