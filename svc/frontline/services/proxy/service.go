package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/services/router"
	"golang.org/x/net/http2"
)

type service struct {
	instanceID        string
	platform          string
	region            string
	apexDomain        string
	clock             clock.Clock
	transport         *http.Transport
	h2cTransport      *http2.Transport
	maxHops           int
	errorPageRenderer errorpage.Renderer
}

var _ Service = (*service)(nil)

func New(cfg Config) (*service, error) {
	maxHops := cfg.MaxHops
	if maxHops == 0 {
		maxHops = 3
	}

	var transport *http.Transport
	if cfg.Transport != nil {
		transport = cfg.Transport
	} else {
		//nolint:exhaustruct
		transport = &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			ForceAttemptHTTP2: true,
			// TCP KeepAlive for detecting dead connections and keeping connections alive through NAT/firewalls
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			// Enable TLS session resumption for faster cross-region forwarding
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				ClientSessionCache: tls.NewLRUClientSessionCache(100),
			},
		}

		if cfg.MaxIdleConns > 0 {
			transport.MaxIdleConns = cfg.MaxIdleConns
		}

		if cfg.IdleConnTimeout > 0 {
			transport.IdleConnTimeout = cfg.IdleConnTimeout
		}

		if cfg.TLSHandshakeTimeout > 0 {
			transport.TLSHandshakeTimeout = cfg.TLSHandshakeTimeout
		}

	}

	// Create h2c transport for HTTP/2 cleartext connections to sentinel
	//nolint:exhaustruct
	h2cTransport := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			// For h2c, we dial plain TCP (not TLS)
			d := net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return d.DialContext(ctx, network, addr)
		},
	}

	renderer := cfg.ErrorPageRenderer
	if renderer == nil {
		renderer = errorpage.NewRenderer()
	}

	return &service{
		instanceID:        cfg.InstanceID,
		platform:          cfg.Platform,
		region:            cfg.Region,
		apexDomain:        cfg.ApexDomain,
		clock:             cfg.Clock,
		transport:         transport,
		h2cTransport:      h2cTransport,
		maxHops:           maxHops,
		errorPageRenderer: renderer,
	}, nil
}

func (s *service) Forward(ctx context.Context, sess *zen.Session, decision router.RouteDecision) error {
	switch decision.Destination {
	case router.DestinationLocalSentinel:
		return s.forwardToSentinel(ctx, sess, decision.Address, decision.DeploymentID)
	case router.DestinationRemoteRegion:
		return s.forwardToRegion(ctx, sess, decision.Address)
	case router.DestinationPortal:
		return s.forwardToPortal(ctx, sess, decision.Address, decision.PathPrefix)
	default:
		return fault.New("unknown route destination: "+string(decision.Destination),
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("unhandled route destination type"),
			fault.Public("Internal routing error"),
		)
	}
}

func (s *service) forwardToSentinel(ctx context.Context, sess *zen.Session, sentinelAddress string, deploymentID string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	targetURL, err := url.Parse(fmt.Sprintf("http://%s", sentinelAddress))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse sentinel URL"),
		)
	}

	return s.forward(ctx, sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeSentinelDirector(sess, deploymentID, startTime),
		destination:  "sentinel",
		transport:    s.h2cTransport,
	})
}

func (s *service) forwardToRegion(ctx context.Context, sess *zen.Session, targetRegionPlatform string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	if hopCountStr := sess.Request().Header.Get(HeaderFrontlineHops); hopCountStr != "" {
		if hops, err := strconv.Atoi(hopCountStr); err == nil {
			proxyHopsTotal.Observe(float64(hops))

			if hops >= s.maxHops {
				logger.Error("too many frontline hops - rejecting request",
					"hops", hops,
					"maxHops", s.maxHops,
					"hostname", sess.Request().Host,
					"requestID", sess.RequestID(),
				)
				return fault.New("too many frontline hops",
					fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
					fault.Internal(fmt.Sprintf("request exceeded maximum hop count: %d", hops)),
					fault.Public("Request routing limit exceeded"),
				)
			}
		}
	}

	targetURL, err := url.Parse(fmt.Sprintf("https://frontline.%s.%s", targetRegionPlatform, s.apexDomain))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse NLB URL"),
		)
	}

	return s.forward(ctx, sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeRegionDirector(sess, startTime),
		destination:  "region",
		transport:    s.transport,
	})
}

// forwardToPortal forwards a request to the portal service. Portal
// routes bypass sentinel entirely — frontline connects directly to the portal
// service over plain HTTP (h2c), similar to how it connects to sentinel.
func (s *service) forwardToPortal(ctx context.Context, sess *zen.Session, portalAddress string, pathPrefix string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	targetURL, err := url.Parse(fmt.Sprintf("http://%s", portalAddress))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse portal URL"),
		)
	}

	return s.forward(ctx, sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makePortalDirector(sess, startTime, pathPrefix),
		destination:  "portal",
		transport:    s.h2cTransport,
	})
}
