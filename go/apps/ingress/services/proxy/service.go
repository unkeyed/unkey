package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type service struct {
	logger     logging.Logger
	ingressID  string
	region     string
	baseDomain string
	clock      clock.Clock
	transport  *http.Transport
	maxHops    int
}

var _ Service = (*service)(nil)

// New creates a new proxy service instance.
func New(cfg Config) (*service, error) {
	// Default MaxHops to 3 if not set
	maxHops := cfg.MaxHops
	if maxHops == 0 {
		maxHops = 3
	}

	// Use shared transport if provided, otherwise create a new one
	var transport *http.Transport
	if cfg.Transport != nil {
		transport = cfg.Transport
	} else {
		// Configure transport with defaults optimized for ingress
		transport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          200, // Higher for ingress workload
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 40 * time.Second, // Longer than gateway timeout (30s) to receive its error response
		}

		// Apply config overrides if provided
		if cfg.MaxIdleConns > 0 {
			transport.MaxIdleConns = cfg.MaxIdleConns
		}

		if cfg.IdleConnTimeout > 0 {
			transport.IdleConnTimeout = cfg.IdleConnTimeout
		}

		if cfg.TLSHandshakeTimeout > 0 {
			transport.TLSHandshakeTimeout = cfg.TLSHandshakeTimeout
		}

		if cfg.ResponseHeaderTimeout > 0 {
			transport.ResponseHeaderTimeout = cfg.ResponseHeaderTimeout
		}
	}

	return &service{
		logger:     cfg.Logger,
		ingressID:  cfg.IngressID,
		region:     cfg.Region,
		baseDomain: cfg.BaseDomain,
		clock:      cfg.Clock,
		transport:  transport,
		maxHops:    maxHops,
	}, nil
}

// ForwardToGateway forwards a request to a local gateway service (HTTP)
// Adds X-Unkey-Deployment-Id header so gateway knows which deployment to route to
func (s *service) ForwardToGateway(ctx context.Context, sess *zen.Session, gateway *db.Gateway, deploymentID string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	targetURL, err := url.Parse(fmt.Sprintf("http://%s", gateway.K8sServiceName))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse gateway URL"),
		)
	}

	return s.forward(sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeGatewayDirector(sess, deploymentID, startTime),
		logTarget:    "gateway",
	})
}

// ForwardToNLB forwards a request to a remote region's NLB (HTTPS)
// Keeps original hostname so remote ingress can do TLS termination and routing
func (s *service) ForwardToNLB(ctx context.Context, sess *zen.Session, targetRegion string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	// Check for too many hops to prevent infinite routing loops
	if hopCountStr := sess.Request().Header.Get(HeaderIngressHops); hopCountStr != "" {
		if hops, err := strconv.Atoi(hopCountStr); err == nil && hops >= s.maxHops {
			s.logger.Error("too many ingress hops - rejecting request",
				"hops", hops,
				"maxHops", s.maxHops,
				"hostname", sess.Request().Host,
				"requestID", sess.RequestID(),
			)
			return fault.New("too many ingress hops",
				fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
				fault.Internal(fmt.Sprintf("request exceeded maximum hop count: %d", hops)),
				fault.Public("Request routing limit exceeded"),
			)
		}
	}

	targetURL, err := url.Parse(fmt.Sprintf("https://%s.%s", targetRegion, s.baseDomain))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse NLB URL"),
		)
	}

	return s.forward(sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeNLBDirector(sess, startTime),
		logTarget:    "NLB",
	})
}
