package handler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/apps/gateway/services/router"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct {
	Logger        logging.Logger
	RouterService router.Service
	Clock         clock.Clock
	Transport     *http.Transport
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
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

// categorizeProxyError determines the appropriate error code and message based on the error type
func categorizeProxyError(err error) (codes.URN, string) {
	// Check for client-side cancellation (client closed connection)
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Gateway.Proxy.GatewayTimeout.URN(),
			"The request took too long to process. Please try again later."
	}

	// Check for network errors
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		// Check for timeout
		if netErr.Timeout() {
			return codes.Gateway.Proxy.GatewayTimeout.URN(),
				"The request took too long to process. Please try again later."
		}

		// Check for connection refused
		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Gateway.Proxy.ServiceUnavailable.URN(),
				"The service is temporarily unavailable. Please try again later."
		}

		// Check for connection reset
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Gateway.Proxy.BadGateway.URN(),
				"Connection was reset by the backend service. Please try again."
		}

		// Check for no route to host
		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Gateway.Proxy.ServiceUnavailable.URN(),
				"The service is unreachable. Please try again later."
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return codes.Gateway.Proxy.ServiceUnavailable.URN(),
				"The service could not be found. Please check your configuration."
		}
		if dnsErr.IsTimeout {
			return codes.Gateway.Proxy.GatewayTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
	}

	// Default to bad gateway
	return codes.Gateway.Proxy.BadGateway.URN(),
		"Unable to connect to the backend service. Please try again in a few moments."
}

func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request()
	startTime := h.Clock.Now()
	var instanceStart, instanceEnd time.Time

	// Always add timing headers when function returns (success or error)
	defer func() {
		gatewayDuration := h.Clock.Now().Sub(startTime)
		sess.ResponseWriter().Header().Set("X-Unkey-Gateway-Time", fmt.Sprintf("%dms", gatewayDuration.Milliseconds()))

		// Add instance timing if we got to the proxy stage
		if !instanceStart.IsZero() && !instanceEnd.IsZero() {
			instanceDuration := instanceEnd.Sub(instanceStart)
			sess.ResponseWriter().Header().Set("X-Unkey-Instance-Time", fmt.Sprintf("%dms", instanceDuration.Milliseconds()))
		}
	}()

	// Get deployment ID from header
	deploymentID := req.Header.Get("X-Deployment-ID")
	if deploymentID == "" {
		return fault.New("missing deployment ID",
			fault.Code(codes.User.BadRequest.MissingRequiredHeader.URN()),
			fault.Internal("X-Deployment-ID header not set"),
			fault.Public("Bad request"),
		)
	}

	// Get deployment to validate it belongs to this environment
	_, err := h.RouterService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return err // Error already has proper fault code from router service
	}

	// Select a healthy instance for this deployment
	instance, err := h.RouterService.SelectInstance(ctx, deploymentID)
	if err != nil {
		return err // Error already has proper fault code from router service
	}

	// Build target URL using instance address
	targetURL, err := url.Parse("http://" + instance.Address)
	if err != nil {
		h.Logger.Error("invalid instance address", "address", instance.Address, "error", err)
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Internal.InvalidConfiguration.URN()),
			fault.Internal("invalid service address"),
			fault.Public("Service configuration error"),
		)
	}

	// Wrap the response writer to capture errors without writing to client
	wrapper := &errorCapturingWriter{
		ResponseWriter: sess.ResponseWriter(),
	}

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(outReq *http.Request) {
			instanceStart = h.Clock.Now()
			outReq.URL.Scheme = targetURL.Scheme
			outReq.URL.Host = targetURL.Host
			outReq.Host = req.Host

			// Copy headers
			if outReq.Header == nil {
				outReq.Header = make(http.Header)
			}

			// Add forwarded headers
			if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
				outReq.Header.Set("X-Forwarded-For", clientIP)
			}
			outReq.Header.Set("X-Forwarded-Host", req.Host)
			outReq.Header.Set("X-Forwarded-Proto", "http")
		},
		Transport: h.Transport,
		ModifyResponse: func(resp *http.Response) error {
			// Record when instance responded
			instanceEnd = h.Clock.Now()
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			// Record when instance failed
			instanceEnd = h.Clock.Now()

			// Capture the error for middleware to handle
			if ecw, ok := w.(*errorCapturingWriter); ok {
				ecw.capturedError = err

				h.Logger.Warn("proxy error",
					"deploymentID", deploymentID,
					"instanceID", instance.ID,
					"target", instance.Address,
					"error", err.Error(),
				)
			}
		},
	}

	h.Logger.Debug("proxying request",
		"method", req.Method,
		"path", req.URL.Path,
		"deploymentID", deploymentID,
		"instanceID", instance.ID,
		"target", instance.Address,
	)

	// Serve the proxied request with wrapped writer
	proxy.ServeHTTP(wrapper, req)

	// If error was captured, return it to middleware for consistent error handling
	if wrapper.capturedError != nil {
		urn, message := categorizeProxyError(wrapper.capturedError)
		return fault.Wrap(wrapper.capturedError,
			fault.Code(urn),
			fault.Internal(fmt.Sprintf("proxy error forwarding to instance %s", instance.Address)),
			fault.Public(message),
		)
	}

	return nil
}
