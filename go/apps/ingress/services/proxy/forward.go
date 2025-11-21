package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// forwardConfig holds configuration for forwarding a request
type forwardConfig struct {
	targetURL    *url.URL
	startTime    time.Time
	directorFunc func(*http.Request)
	logTarget    string // Description for logging (e.g., "gateway", "NLB")
}

// forward handles the common logic for proxying requests to either gateway or NLB
func (s *service) forward(sess *zen.Session, cfg forwardConfig) error {
	// Set response headers BACK TO CLIENT so they can see which ingress handled their request
	// These are useful for debugging and support tickets
	sess.ResponseWriter().Header().Set(HeaderIngressID, s.ingressID)
	sess.ResponseWriter().Header().Set(HeaderRegion, s.region)
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	var proxyStartTime time.Time

	// Always add timing headers when function returns (success or error)
	defer func() {
		totalTime := s.clock.Now().Sub(cfg.startTime)
		if !proxyStartTime.IsZero() {
			sess.ResponseWriter().Header().Set("X-Unkey-Ingress-Time", fmt.Sprintf("%dms", proxyStartTime.Sub(cfg.startTime).Milliseconds()))
		}
		sess.ResponseWriter().Header().Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))
	}()

	// Wrap the response writer to capture errors without writing to client
	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())

	// Create reverse proxy with shared transport
	proxy := &httputil.ReverseProxy{
		Transport: s.transport,
		Director: func(req *http.Request) {
			// Record when we start calling downstream service
			proxyStartTime = s.clock.Now()

			// Update URL to target
			req.URL.Scheme = cfg.targetURL.Scheme
			req.URL.Host = cfg.targetURL.Host

			// Call the specific director function for additional configuration
			cfg.directorFunc(req)
		},
		ModifyResponse: func(resp *http.Response) error {
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			// Capture the error for middleware to handle
			if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
				ecw.SetError(err)

				s.logger.Warn(fmt.Sprintf("proxy error forwarding to %s", cfg.logTarget),
					"error", err.Error(),
					"target", cfg.targetURL.String(),
					"hostname", r.Host,
				)
			}
		},
	}

	// Proxy the request with wrapped writer
	proxy.ServeHTTP(wrapper, sess.Request())

	// If error was captured, return it to middleware for consistent error handling
	if err := wrapper.Error(); err != nil {
		urn, message := categorizeProxyError(err)
		return fault.Wrap(err,
			fault.Code(urn),
			fault.Internal(fmt.Sprintf("proxy error forwarding to %s %s", cfg.logTarget, cfg.targetURL.String())),
			fault.Public(message),
		)
	}

	return nil
}
