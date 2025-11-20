package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"syscall"
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

// categorizeProxyError determines the appropriate error code and message based on the error type
func categorizeProxyError(err error) (codes.URN, string) {
	// Check for client-side cancellation (client closed connection)
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Ingress.Proxy.GatewayTimeout.URN(),
			"The request took too long to process. Please try again later."
	}

	// Check for network errors
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		// Check for timeout
		if netErr.Timeout() {
			return codes.Ingress.Proxy.GatewayTimeout.URN(),
				"The request took too long to process. Please try again later."
		}

		// Check for connection refused
		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Ingress.Proxy.ServiceUnavailable.URN(),
				"The service is temporarily unavailable. Please try again later."
		}

		// Check for connection reset
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Ingress.Proxy.BadGateway.URN(),
				"Connection was reset by the backend service. Please try again."
		}

		// Check for no route to host
		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Ingress.Proxy.ServiceUnavailable.URN(),
				"The service is unreachable. Please try again later."
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return codes.Ingress.Proxy.ServiceUnavailable.URN(),
				"The service could not be found. Please check your configuration."
		}
		if dnsErr.IsTimeout {
			return codes.Ingress.Proxy.GatewayTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
	}

	// Default to bad gateway
	return codes.Ingress.Proxy.BadGateway.URN(),
		"Unable to connect to the backend service. Please try again in a few moments."
}

// ForwardToGateway forwards a request to a local gateway service (HTTP)
// Adds X-Unkey-Deployment-Id header so gateway knows which deployment to route to
func (s *service) ForwardToGateway(ctx context.Context, sess *zen.Session, gateway *db.Gateway, deploymentID string, startTime time.Time) error {
	targetURL, err := url.Parse(fmt.Sprintf("http://%s", gateway.K8sServiceName))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse gateway URL"),
		)
	}

	s.logger.Info("forwarding to local gateway",
		"target", targetURL.String(),
		"gatewayID", gateway.ID,
		"deploymentID", deploymentID,
		"region", gateway.Region,
	)

	return s.forwardToGateway(ctx, sess, targetURL, deploymentID, startTime)
}

// ForwardToNLB forwards a request to a remote region's NLB (HTTPS)
// Keeps original hostname so remote ingress can do TLS termination and routing
func (s *service) ForwardToNLB(ctx context.Context, sess *zen.Session, targetRegion string, startTime time.Time) error {
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

	s.logger.Info("forwarding to remote NLB",
		"target", targetURL.String(),
		"targetRegion", targetRegion,
		"hostname", sess.Request().Host,
	)

	return s.forwardToNLB(ctx, sess, targetURL, startTime)
}

// errorCapturingWriter wraps a ResponseWriter to capture proxy errors
// without writing them to the client. This allows errors to be returned
// to the middleware for consistent error handling.
type errorCapturingWriter struct {
	http.ResponseWriter
	capturedError error
	headerWritten bool
}

func (w *errorCapturingWriter) WriteHeader(statusCode int) {
	if w.capturedError != nil {
		// Discard header writes if we captured an error
		w.headerWritten = true
		return
	}
	w.ResponseWriter.WriteHeader(statusCode)
	w.headerWritten = true
}

func (w *errorCapturingWriter) Write(b []byte) (int, error) {
	if w.capturedError != nil {
		// Discard body writes if we captured an error
		return len(b), nil
	}
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// forwardToGateway handles proxying to a local gateway service
func (s *service) forwardToGateway(_ctx context.Context, sess *zen.Session, targetURL *url.URL, deploymentID string, startTime time.Time) error {
	// Set response headers BACK TO CLIENT so they can see which ingress handled their request
	// These are useful for debugging and support tickets
	sess.ResponseWriter().Header().Set(HeaderIngressID, s.ingressID)
	sess.ResponseWriter().Header().Set(HeaderRegion, s.region)
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	var gatewayStartTime time.Time

	// Wrap the response writer to capture errors without writing to client
	wrapper := &errorCapturingWriter{
		ResponseWriter: sess.ResponseWriter(),
	}

	// Create reverse proxy with shared transport
	proxy := &httputil.ReverseProxy{
		Transport: s.transport,
		Director: func(req *http.Request) {
			// Record when we start calling gateway
			gatewayStartTime = s.clock.Now()

			// Update URL to target
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host

			// Add metadata headers TO DOWNSTREAM SERVICE (gateway)
			// These tell the gateway which ingress forwarded the request
			req.Header.Set(HeaderIngressID, s.ingressID)
			req.Header.Set(HeaderRegion, s.region)
			req.Header.Set(HeaderRequestID, sess.RequestID())

			// Add timing to track latency added by this ingress (routing overhead)
			ingressRoutingTimeMs := gatewayStartTime.Sub(startTime).Milliseconds()
			req.Header.Set(HeaderIngressTime, strconv.FormatInt(ingressRoutingTimeMs, 10))

			// Add standard proxy headers for local gateway
			req.Header.Set(HeaderForwardedProto, "https")

			// Add deployment ID so gateway knows which deployment to route to
			req.Header.Set(HeaderDeploymentID, deploymentID)
		},
		ModifyResponse: func(resp *http.Response) error {
			// Calculate total time
			totalTime := s.clock.Now().Sub(startTime)

			// Add timing headers to response
			resp.Header.Set("X-Unkey-Ingress-Time", fmt.Sprintf("%dms", gatewayStartTime.Sub(startTime).Milliseconds()))
			resp.Header.Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			// Capture the error for middleware to handle
			if ecw, ok := w.(*errorCapturingWriter); ok {
				ecw.capturedError = err

				s.logger.Warn("proxy error forwarding to gateway",
					"error", err.Error(),
					"target", targetURL.String(),
				)
			}
		},
	}

	// Proxy the request with wrapped writer
	proxy.ServeHTTP(wrapper, sess.Request())

	// If error was captured, return it to middleware for consistent error handling
	if wrapper.capturedError != nil {
		// Add timing headers even on error
		totalTime := s.clock.Now().Sub(startTime)
		sess.ResponseWriter().Header().Set("X-Unkey-Ingress-Time", fmt.Sprintf("%dms", gatewayStartTime.Sub(startTime).Milliseconds()))
		sess.ResponseWriter().Header().Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))

		urn, message := categorizeProxyError(wrapper.capturedError)
		return fault.Wrap(wrapper.capturedError,
			fault.Code(urn),
			fault.Internal(fmt.Sprintf("proxy error forwarding to gateway %s", targetURL.String())),
			fault.Public(message),
		)
	}

	return nil
}

// forwardToNLB handles proxying to a remote region's NLB
func (s *service) forwardToNLB(_ctx context.Context, sess *zen.Session, targetURL *url.URL, startTime time.Time) error {
	// Set response headers BACK TO CLIENT so they can see which ingress handled their request
	// These are useful for debugging and support tickets
	sess.ResponseWriter().Header().Set(HeaderIngressID, s.ingressID)
	sess.ResponseWriter().Header().Set(HeaderRegion, s.region)
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	var nlbStartTime time.Time

	// Wrap the response writer to capture errors without writing to client
	wrapper := &errorCapturingWriter{
		ResponseWriter: sess.ResponseWriter(),
	}

	// Create reverse proxy with shared transport
	proxy := &httputil.ReverseProxy{
		Transport: s.transport,
		Director: func(req *http.Request) {
			// Record when we start calling NLB
			nlbStartTime = s.clock.Now()

			// Update URL to target
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host

			// Add metadata headers TO DOWNSTREAM SERVICE (remote ingress)
			// These tell the remote ingress which ingress forwarded the request
			req.Header.Set(HeaderIngressID, s.ingressID)
			req.Header.Set(HeaderRegion, s.region)
			req.Header.Set(HeaderRequestID, sess.RequestID())

			// Add timing to track latency added by this ingress (routing overhead)
			ingressRoutingTimeMs := nlbStartTime.Sub(startTime).Milliseconds()
			req.Header.Set(HeaderIngressTime, strconv.FormatInt(ingressRoutingTimeMs, 10))

			// Remote ingress - preserve original Host for TLS termination and routing
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
		},
		ModifyResponse: func(resp *http.Response) error {
			// Calculate total time
			totalTime := s.clock.Now().Sub(startTime)

			// Add timing headers to response
			resp.Header.Set("X-Unkey-Ingress-Time", fmt.Sprintf("%dms", nlbStartTime.Sub(startTime).Milliseconds()))
			resp.Header.Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			// Capture the error for middleware to handle
			if ecw, ok := w.(*errorCapturingWriter); ok {
				ecw.capturedError = err

				s.logger.Warn("proxy error forwarding to NLB",
					"error", err.Error(),
					"target", targetURL.String(),
					"hostname", r.Host,
				)
			}
		},
	}

	// Proxy the request with wrapped writer
	proxy.ServeHTTP(wrapper, sess.Request())

	// If error was captured, return it to middleware for consistent error handling
	if wrapper.capturedError != nil {
		// Add timing headers even on error
		totalTime := s.clock.Now().Sub(startTime)
		sess.ResponseWriter().Header().Set("X-Unkey-Ingress-Time", fmt.Sprintf("%dms", nlbStartTime.Sub(startTime).Milliseconds()))
		sess.ResponseWriter().Header().Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))

		urn, message := categorizeProxyError(wrapper.capturedError)
		return fault.Wrap(wrapper.capturedError,
			fault.Code(urn),
			fault.Internal(fmt.Sprintf("proxy error forwarding to NLB %s", targetURL.String())),
			fault.Public(message),
		)
	}

	return nil
}
