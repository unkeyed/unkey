// Package ctrlclient builds preflight's ConnectRPC clients for the
// control-plane API. Every tier-1+ probe that needs to hit ctrl goes
// through here so transport and auth configuration stay in exactly
// one place.
//
// The underlying http.Client uses h2c so the harness's plain-http
// ctrl server works, and the same transport speaks HTTP/2 + TLS to
// real staging/prod without any branching.
package ctrlclient

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"

	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// defaultTimeout covers CreateDeployment RPCs against a cold ctrl API.
// Individual probes that need a tighter budget should wrap the returned
// client's calls in their own context.WithTimeout.
const defaultTimeout = 30 * time.Second

// NewDeployClient returns a DeployServiceClient pointed at
// env.CtrlBaseURL. Use for CreateDeployment, GetDeployment, and every
// other DeployService RPC. Callers set `Authorization: Bearer <token>`
// on each connect.Request via the returned helper from AuthHeader.
func NewDeployClient(env *core.Env) ctrlv1connect.DeployServiceClient {
	return ctrlv1connect.NewDeployServiceClient(newHTTPClient(), env.CtrlBaseURL)
}

// AuthHeader returns the value to set on connect.Request.Header() for
// `Authorization`. Thin wrapper; exists so probe call sites do not
// re-stringify the "Bearer " prefix.
func AuthHeader(env *core.Env) string {
	return "Bearer " + env.CtrlAuthToken
}

// newHTTPClient builds the h2c-capable HTTP client every ctrl RPC
// shares. Centralised so transport timeouts, keepalive, and the
// DialTLSContext plumbing stay consistent across probes.
func newHTTPClient() *http.Client {
	//nolint:exhaustruct // default http2.Transport settings are correct for a one-shot probe
	return &http.Client{
		Timeout: defaultTimeout,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
			ReadIdleTimeout: 10 * time.Second,
			PingTimeout:     5 * time.Second,
		},
	}
}
