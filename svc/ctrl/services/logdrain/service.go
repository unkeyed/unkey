// Package logdrain verifies customer log drain destinations. The dashboard
// cannot probe an endpoint itself because the SSRF guard that keeps
// deliveries away from loopback, private, and link-local addresses lives in
// Go, alongside the sink that does the real deliveries.
package logdrain

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/ssrf"
)

// probeTimeout bounds the whole attempt. The dashboard blocks on this call
// while it creates or updates a drain.
const probeTimeout = 10 * time.Second

// Service implements [ctrlv1connect.LogdrainServiceHandler].
type Service struct {
	ctrlv1connect.UnimplementedLogdrainServiceHandler
	bearer string

	// ssrfOpts configure both client and every [ssrf.ValidateEndpoint] call,
	// which needs the same guards the transport applies.
	ssrfOpts []ssrf.Option
	client   *http.Client
}

// Config configures a [Service].
type Config struct {
	// Bearer is the preshared token callers must provide.
	Bearer string

	// UnsafeAllowPrivateEndpoints permits private, plain-http endpoints.
	// Development only.
	UnsafeAllowPrivateEndpoints bool
}

// New builds the probe client once. [ssrf.New] opens a transport per call and
// nothing closes it, so building one per request leaks connections.
func New(cfg Config) *Service {
	opts := []ssrf.Option{ssrf.WithTimeout(probeTimeout)}
	if cfg.UnsafeAllowPrivateEndpoints {
		opts = append(opts, ssrf.UnsafeAllowAll())
	}
	return &Service{
		UnimplementedLogdrainServiceHandler: ctrlv1connect.UnimplementedLogdrainServiceHandler{},
		bearer:                              cfg.Bearer,
		ssrfOpts:                            opts,
		client:                              ssrf.New(opts...),
	}
}

var _ ctrlv1connect.LogdrainServiceHandler = (*Service)(nil)
