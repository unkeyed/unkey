package controlplane

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

// connectInterceptor creates a streaming-compatible interceptor that adds authentication
// and metadata headers to all requests (both unary and streaming).
//
// The interceptor automatically injects:
// - Authorization header with Bearer token
// - X-Krane-Region header for routing
// - X-Krane-Cluster-Id header for routing
//
// This is the recommended way to create interceptors for control plane clients.
func connectInterceptor(region, clusterID, bearer string) connect.Interceptor {
	return &authInterceptor{
		region:    region,
		clusterID: clusterID,
		bearer:    bearer,
	}
}

// authInterceptor implements connect.Interceptor for both unary and streaming calls.
//
// It automatically adds authentication and routing metadata to all outgoing requests.
// The interceptor is stateless and safe for concurrent use.
type authInterceptor struct {
	region    string
	clusterID string
	bearer    string
}

// WrapUnary intercepts unary RPC calls by adding required headers before forwarding
// the request to the next handler.
//
// This method adds authentication and routing headers to all unary requests.
// The headers are set before the request is processed by the actual service.
func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
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

// WrapStreamingHandler intercepts streaming server calls.
//
// For client-side interceptors, this method is typically not invoked but
// is implemented to satisfy the [connect.Interceptor] interface.
// The implementation simply forwards the call without modification.
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

// streamingClientInterceptor wraps streaming client connections to add headers
// on the first message send.
//
// This type is needed because streaming connections don't allow header modification
// after establishment. Headers must be injected with the first message payload.
type streamingClientInterceptor struct {
	connect.StreamingClientConn
	interceptor *authInterceptor
	spec        connect.Spec
	headersSet  bool
}

// Send intercepts outgoing messages in streaming calls by adding authentication
// and routing headers on the first message.
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
// by the Send() method to inject authentication and routing headers.
func (s *streamingClientInterceptor) RequestHeader() http.Header {
	return s.StreamingClientConn.RequestHeader()
}

// setHeaders adds the required authentication and metadata headers to the
// provided header map.
//
// This method injects:
// - X-Krane-Region: The client's geographical region
// - Authorization: Bearer token for authentication
//
// All headers use Set() to overwrite any existing values.
func (i *authInterceptor) setHeaders(header http.Header) {
	header.Set("X-Krane-Region", i.region)
	header.Set("X-Krane-Cluster-Id", i.clusterID)
	header.Set("Authorization", fmt.Sprintf("Bearer %s", i.bearer))
}
