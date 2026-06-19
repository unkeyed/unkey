package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
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

	// regionHeader is the precomputed "platform::region" value written to
	// the X-Unkey-Region response header on every request.
	regionHeader string

	// copyBufs feeds httputil.ReverseProxy.BufferPool. Without a pool the
	// proxy allocates a fresh 32 KiB copy buffer per request, which was
	// two thirds of all allocated bytes on the hot path.
	copyBufs copyBufferPool

	// reverseProxy is built once and shared by every request. Per-request
	// state travels through the request context as a [proxyState] instead of
	// rebuilding the proxy and its callbacks per request. See [proxyState].
	reverseProxy *httputil.ReverseProxy
}

// copyBufferPool implements httputil.BufferPool over a sync.Pool. Buffers
// are stored as pointers so Get/Put do not allocate on the interface
// boxing.
type copyBufferPool struct{ p sync.Pool }

func (p *copyBufferPool) Get() []byte {
	if b, ok := p.p.Get().(*[]byte); ok {
		return *b
	}
	return make([]byte, 32<<10)
}

func (p *copyBufferPool) Put(b []byte) { p.p.Put(&b) }

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
			DialContext: countingDialContext((&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext, destinationFrontline), //nolint:exhaustruct
			// The peer-region hop targets one frontline endpoint per
			// region, so per-host is effectively the pool size.
			MaxIdleConns:          1024,
			MaxIdleConnsPerHost:   512,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			// Pass client encodings through untouched; never negotiate
			// and undo gzip on the client's behalf.
			DisableCompression: true,
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

	s := &service{
		instanceID:         cfg.InstanceID,
		platform:           cfg.Platform,
		region:             cfg.Region,
		apexDomain:         cfg.ApexDomain,
		clock:              cfg.Clock,
		transport:          transport,
		upstreamTransports: upstreamTransports,
		maxHops:            maxHops,
		errorPageRenderer:  renderer,
		regionHeader:       cfg.Platform + "::" + cfg.Region,
		copyBufs:           copyBufferPool{p: sync.Pool{New: nil}},
		reverseProxy:       nil,
	}
	// Built after the struct exists: the proxy's callbacks are method values
	// bound to s and its BufferPool points at s.copyBufs.
	s.reverseProxy = s.newReverseProxy()
	return s, nil
}

func (s *service) ForwardToInstance(ctx context.Context, sess *zen.Session, protocol db.DeploymentsUpstreamProtocol, instance db.FindInstancesByDeploymentIDRow) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	// Build the URL directly instead of url.Parse(fmt.Sprintf(...)): the
	// address is host:port straight from the DB, there is nothing to parse,
	// and this runs on every request.
	targetURL := &url.URL{Scheme: "http", Host: instance.Address} //nolint:exhaustruct

	transport := s.upstreamTransports.Get(protocol)

	return s.forward(ctx, sess, forwardConfig{
		targetURL:    targetURL,
		startTime:    startTime,
		directorFunc: s.makeInstanceDirector(sess, startTime),
		destination:  destinationInstance,
		transport:    transport,
	})
}

func (s *service) ForwardToRegion(ctx context.Context, sess *zen.Session, targetRegionPlatform string) error {
	startTime, _ := RequestStartTimeFromContext(ctx)

	if hopCountStr := sess.Request().Header.Get(HeaderFrontlineHops); hopCountStr != "" {
		if hops, err := strconv.Atoi(hopCountStr); err == nil {
			srcRegion := sess.Request().Header.Get(HeaderRegion)
			if srcRegion == "" {
				srcRegion = s.regionHeader
			}
			hopsHistogram.WithLabelValues(srcRegion, targetRegionPlatform).Observe(float64(hops))

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
		destination:  destinationFrontline,
		transport:    s.transport,
	})
}
