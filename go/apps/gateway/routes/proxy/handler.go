package handler

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
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

func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request()
	startTime := h.Clock.Now()

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

	// Record gateway overhead (time to validate deployment + select instance)
	gatewayTime := h.Clock.Now()
	gatewayDuration := gatewayTime.Sub(startTime)

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

	// Track instance response time
	var instanceStart, instanceEnd time.Time

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
			instanceEnd = h.Clock.Now()
			instanceDuration := instanceEnd.Sub(instanceStart)

			// Add timing headers (gateway + instance only, not total - ingress handles that)
			resp.Header.Set("X-Unkey-Gateway-Time", fmt.Sprintf("%dms", gatewayDuration.Milliseconds()))
			resp.Header.Set("X-Unkey-Instance-Time", fmt.Sprintf("%dms", instanceDuration.Milliseconds()))

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.Logger.Error("proxy error",
				"deploymentID", deploymentID,
				"instanceID", instance.ID,
				"target", instance.Address,
				"error", err,
			)
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, "Bad Gateway")
		},
	}

	h.Logger.Debug("proxying request",
		"method", req.Method,
		"path", req.URL.Path,
		"deploymentID", deploymentID,
		"instanceID", instance.ID,
		"target", instance.Address,
	)

	// Serve the proxied request
	proxy.ServeHTTP(sess.ResponseWriter(), req)
	return nil
}
