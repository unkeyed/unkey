package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type service struct {
	logger     logging.Logger
	ingressID  string
	region     string
	baseDomain string
	clock      interface{ Now() time.Time }
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
			ResponseHeaderTimeout: 30 * time.Second,
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

// GetMaxHops returns the maximum number of ingress hops allowed
func (s *service) GetMaxHops() int {
	return s.maxHops
}

// writeProxyError writes a clean JSON or HTML error response for proxy failures
func (s *service) writeProxyError(w http.ResponseWriter, r *http.Request, requestID string, err error) {
	s.logger.Error("proxy error",
		"error", err.Error(),
		"requestID", requestID,
		"host", r.Host,
		"path", r.URL.Path,
	)

	// Set response headers so client knows which ingress hit the error
	w.Header().Set(HeaderIngressID, s.ingressID)
	w.Header().Set(HeaderRegion, s.region)
	w.Header().Set(HeaderRequestID, requestID)

	// Determine response format based on Accept header
	acceptHeader := r.Header.Get("Accept")
	preferJSON := strings.Contains(acceptHeader, "application/json") ||
		strings.Contains(acceptHeader, "application/*") ||
		(strings.Contains(acceptHeader, "*/*") && !strings.Contains(acceptHeader, "text/html"))

	status := http.StatusBadGateway
	code := codes.Ingress.Proxy.BadGateway.URN()
	message := "Unable to reach service"

	if preferJSON {
		// Write JSON error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    string(code),
				"message": message,
				"status":  status,
			},
		})
	} else {
		// Write HTML error
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>502 Bad Gateway</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        .error-code { color: #999; font-size: 0.9em; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>502 Bad Gateway</h1>
    <p>%s</p>
    <p>Please try again in a moment. If the problem persists, contact support.</p>
    <p class="error-code">Error: %s</p>
    <p class="error-code">Request ID: %s</p>
</body>
</html>`, message, code, requestID)
		w.Write([]byte(html))
	}
}

// ForwardToLocal forwards a request to a local gateway service (HTTP)
func (s *service) ForwardToLocal(ctx context.Context, sess *zen.Session, deployment *partitionv1.Deployment, startTime time.Time) error {
	targetURL, err := url.Parse(fmt.Sprintf("http://%s", deployment.K8SServiceName))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse gateway URL"),
		)
	}

	s.logger.Info("forwarding to local gateway",
		"target", targetURL.String(),
		"deployment", deployment.Id,
		"region", deployment.Region,
	)

	return s.forward(ctx, sess, targetURL, deployment, true, startTime)
}

// ForwardToRemote forwards a request to a remote ingress (HTTPS)
func (s *service) ForwardToRemote(ctx context.Context, sess *zen.Session, targetRegion string, deployment *partitionv1.Deployment, startTime time.Time) error {
	targetURL, err := url.Parse(fmt.Sprintf("https://%s.%s", targetRegion, s.baseDomain))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse remote ingress URL"),
		)
	}

	s.logger.Info("forwarding to remote ingress",
		"target", targetURL.String(),
		"targetRegion", targetRegion,
		"deploymentRegion", deployment.Region,
	)

	return s.forward(ctx, sess, targetURL, deployment, false, startTime)
}

// forward handles the actual proxying with shared transport
func (s *service) forward(ctx context.Context, sess *zen.Session, targetURL *url.URL, deployment *partitionv1.Deployment, isLocal bool, startTime time.Time) error {
	// Set response headers BACK TO CLIENT so they can see which ingress handled their request
	// These are useful for debugging and support tickets
	sess.ResponseWriter().Header().Set(HeaderIngressID, s.ingressID)
	sess.ResponseWriter().Header().Set(HeaderRegion, s.region)
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	// Create reverse proxy with shared transport
	proxy := &httputil.ReverseProxy{
		Transport: s.transport,
		Director: func(req *http.Request) {
			// Update URL to target
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host

			// Add metadata headers TO DOWNSTREAM SERVICE (gateway/remote ingress)
			// These tell the downstream service which ingress forwarded the request
			req.Header.Set(HeaderIngressID, s.ingressID)
			req.Header.Set(HeaderRegion, s.region)
			req.Header.Set(HeaderRequestID, sess.RequestID())

			// Add timing to track latency added by this ingress
			ingressTimeMs := s.clock.Now().Sub(startTime).Milliseconds()
			req.Header.Set(HeaderIngressTime, strconv.FormatInt(ingressTimeMs, 10))

			if isLocal {
				// Local gateway - add standard proxy headers
				req.Header.Set(HeaderForwardedProto, "https")
				req.Header.Set(HeaderDeploymentID, deployment.Id)
			} else {
				// Remote ingress - preserve original Host for routing
				req.Host = sess.Request().Host
				req.Header.Set("Host", sess.Request().Host)

				// Add parent tracking to trace the forwarding chain
				req.Header.Set(HeaderParentIngressID, s.ingressID)
				req.Header.Set(HeaderParentRequestID, sess.RequestID())

				// Parse and increment hop count to prevent infinite loops
				currentHops := 0
				if hopCountStr := req.Header.Get(HeaderIngressHops); hopCountStr != "" {
					if parsed, err := strconv.Atoi(hopCountStr); err == nil {
						currentHops = parsed
					}
				}
				currentHops++
				req.Header.Set(HeaderIngressHops, strconv.Itoa(currentHops))

				// Log warning if approaching max hops
				if currentHops >= s.maxHops-1 {
					s.logger.Warn("approaching max hops limit",
						"currentHops", currentHops,
						"maxHops", s.maxHops,
						"hostname", req.Host,
					)
				}
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			s.writeProxyError(w, r, sess.RequestID(), err)
		},
	}

	// Proxy the request
	proxy.ServeHTTP(sess.ResponseWriter(), sess.Request())

	return nil
}
