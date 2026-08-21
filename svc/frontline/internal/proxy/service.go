package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
)

// frontlineMetadataTTL limits replay of a captured peer request. A peer
// verifies the token when it receives the request headers, before it reads a
// streaming request body.
const frontlineMetadataTTL = time.Minute

type service struct {
	instanceID         string
	platform           string
	region             string
	apexDomain         string
	clock              clock.Clock
	transport          *http.Transport
	upstreamTransports *TransportRegistry
	maxHops            int
	metadata           *meta.Codec
	errorPageRenderer  errorpage.Renderer
}

var _ Service = (*service)(nil)

func New(cfg Config) (*service, error) {
	if err := assert.All(
		assert.NotNilAndNotZero(cfg.Metadata, "metadata codec is required"),
		assert.GreaterOrEqual(cfg.MaxHops, 0, "max hops must not be negative"),
	); err != nil {
		return nil, fmt.Errorf("invalid proxy config: %w", err)
	}

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
		metadata:           cfg.Metadata,
		errorPageRenderer:  renderer,
	}, nil
}

func (s *service) ForwardToInstance(ctx context.Context, sess *zen.Session, protocol db.DeploymentsUpstreamProtocol, instance db.FindInstancesByDeploymentIDRow) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	targetURL, err := url.Parse(fmt.Sprintf("http://%s", instance.Address))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse instance URL"),
		)
	}

	transport := s.upstreamTransports.Get(protocol)

	return s.forward(ctx, sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeInstanceDirector(sess, startTime),
		destination:  destinationInstance,
		transport:    transport,
	})
}

func (s *service) ForwardToRegion(ctx context.Context, sess *zen.Session, targetRegionPlatform string, hops []meta.Hop) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	srcRegion := fmt.Sprintf("%s::%s", s.platform, s.region)
	hopCount := len(hops)
	hopsHistogram.WithLabelValues(srcRegion, targetRegionPlatform).Observe(float64(hopCount))
	if hopCount >= s.maxHops {
		logger.Error("too many frontline hops - rejecting request",
			"hops", hopCount,
			"maxHops", s.maxHops,
			"hostname", sess.Request().Host,
			"requestID", sess.RequestID(),
		)
		return fault.New("too many frontline hops",
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal(fmt.Sprintf("request exceeded maximum hop count: %d", hopCount)),
			fault.Public("Request routing limit exceeded"),
		)
	}

	now := s.clock.Now()
	hops = append(hops, meta.Hop{
		Region:        srcRegion,
		RequestID:     sess.RequestID(),
		FrontlineID:   s.instanceID,
		TimeUnixMilli: now.UnixMilli(),
	})

	metadata := &meta.Metadata{
		//nolint:exhaustruct // Frontline metadata only needs an expiry.
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: now.Add(frontlineMetadataTTL).Unix(),
		},
		Hops: hops,
	}
	signedMetadata, err := s.metadata.Marshal(metadata)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to marshal Frontline metadata"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	if len(metadata.Hops)+1 >= s.maxHops {
		logger.Warn("approaching max hops limit",
			"currentHops", len(metadata.Hops),
			"maxHops", s.maxHops,
			"hostname", sess.Request().Host,
		)
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
		directorFunc: s.makeRegionDirector(sess, startTime, signedMetadata),
		destination:  destinationFrontline,
		transport:    s.transport,
	})
}
