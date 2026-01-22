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
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/zen"
	"golang.org/x/net/http2"
)

type service struct {
	logger       logging.Logger
	frontlineID  string
	region       string
	apexDomain   string
	clock        clock.Clock
	transport    *http.Transport
	h2cTransport *http2.Transport
	maxHops      int
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
			ResponseHeaderTimeout: 40 * time.Second, // Longer than sentinel timeout (30s) to receive its error response
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

		if cfg.ResponseHeaderTimeout > 0 {
			transport.ResponseHeaderTimeout = cfg.ResponseHeaderTimeout
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

	return &service{
		logger:       cfg.Logger,
		frontlineID:  cfg.FrontlineID,
		region:       cfg.Region,
		apexDomain:   cfg.ApexDomain,
		clock:        cfg.Clock,
		transport:    transport,
		h2cTransport: h2cTransport,
		maxHops:      maxHops,
	}, nil
}

func (s *service) ForwardToSentinel(ctx context.Context, sess *zen.Session, sentinel *db.Sentinel, deploymentID string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	targetURL, err := url.Parse(fmt.Sprintf("http://%s", sentinel.K8sAddress))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse sentinel URL"),
		)
	}

	return s.forward(sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeSentinelDirector(sess, deploymentID, startTime),
		logTarget:    "sentinel",
		transport:    s.h2cTransport,
	})
}

func (s *service) ForwardToRegion(ctx context.Context, sess *zen.Session, targetRegion string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	if hopCountStr := sess.Request().Header.Get(HeaderFrontlineHops); hopCountStr != "" {
		if hops, err := strconv.Atoi(hopCountStr); err == nil && hops >= s.maxHops {
			s.logger.Error("too many frontline hops - rejecting request",
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

	targetURL, err := url.Parse(fmt.Sprintf("https://frontline.%s.%s", targetRegion, s.apexDomain))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse NLB URL"),
		)
	}

	return s.forward(sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeRegionDirector(sess, startTime),
		logTarget:    "region",
		transport:    s.transport,
	})
}
