package handler

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/go/apps/gateway/services/router"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct {
	Logger        logging.Logger
	RouterService router.Service
	Clock         interface{ Now() time.Time }
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

	// Get deployment ID from header
	deploymentID := req.Header.Get("X-Deployment-ID")
	if deploymentID == "" {
		return fault.New("missing deployment ID",
			fault.Internal("X-Deployment-ID header not set"),
			fault.Public("Bad request"),
		)
	}

	// Get deployment config including target address and middlewares
	deployment, err := h.RouterService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("failed to resolve deployment"),
			fault.Public("Service not available"),
		)
	}

	// Build target URL
	targetURL, err := url.Parse("http://" + deployment.TargetAddress)
	if err != nil {
		h.Logger.Error("invalid target address", "address", deployment.TargetAddress, "error", err)
		return fault.Wrap(err,
			fault.Internal("invalid service address"),
			fault.Public("Service configuration error"),
		)
	}

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(outReq *http.Request) {
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
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.Logger.Error("proxy error",
				"deploymentID", deploymentID,
				"target", deployment.TargetAddress,
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
		"target", deployment.TargetAddress,
	)

	// Serve the proxied request
	proxy.ServeHTTP(sess.ResponseWriter(), req)
	return nil
}
