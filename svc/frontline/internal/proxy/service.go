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
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
)

type service struct {
	instanceID         string
	platform           string
	region             string
	apexDomain         string
	clock              clock.Clock
	transport          *http.Transport
	upstreamTransports *TransportRegistry
	maxHops            int
	errorPageRenderer  errorpage.Renderer
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

	upstreamTransports := cfg.UpstreamTransports
	if upstreamTransports == nil {
		upstreamTransports = NewTransportRegistry()
	}

	renderer := cfg.ErrorPageRenderer
	if renderer == nil {
		renderer = errorpage.NewRenderer()
	}

	return &service{
		instanceID:         cfg.InstanceID,
		platform:           cfg.Platform,
		region:             cfg.Region,
		apexDomain:         cfg.ApexDomain,
		clock:              cfg.Clock,
		transport:          transport,
		upstreamTransports: upstreamTransports,
		maxHops:            maxHops,
		errorPageRenderer:  renderer,
	}, nil
}

func (s *service) Forward(ctx context.Context, sess *zen.Session, decision router.RouteDecision) error {
	if decision.Destination == router.DestinationLocalInstance {
		return s.forwardToInstance(ctx, sess, decision)
	}
	return s.forwardToRegion(ctx, sess, decision.Address)
}

func (s *service) forwardToInstance(ctx context.Context, sess *zen.Session, decision router.RouteDecision) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	targetURL, err := url.Parse(fmt.Sprintf("http://%s", decision.Instance.Address))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse instance URL"),
		)
	}

	transport := s.upstreamTransports.Get(decision.UpstreamProtocol)

	return s.forward(ctx, sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeInstanceDirector(sess, startTime),
		destination:  "instance",
		transport:    transport,
	})
}

func (s *service) forwardToRegion(ctx context.Context, sess *zen.Session, targetRegionPlatform string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	if hopCountStr := sess.Request().Header.Get(HeaderFrontlineHops); hopCountStr != "" {
		if hops, err := strconv.Atoi(hopCountStr); err == nil {
			srcRegion := sess.Request().Header.Get(HeaderRegion)
			if srcRegion == "" {
				srcRegion = fmt.Sprintf("%s::%s", s.platform, s.region)
			}
			proxyHops.WithLabelValues(srcRegion, targetRegionPlatform).Observe(float64(hops))

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
