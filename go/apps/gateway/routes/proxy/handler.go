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

func categorizeProxyError(err error) (codes.URN, string) {
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Gateway.Proxy.GatewayTimeout.URN(),
			"The request took too long to process. Please try again later."
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return codes.Gateway.Proxy.GatewayTimeout.URN(),
				"The request took too long to process. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Gateway.Proxy.ServiceUnavailable.URN(),
				"The service is temporarily unavailable. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Gateway.Proxy.BadGateway.URN(),
				"Connection was reset by the backend service. Please try again."
		}

		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Gateway.Proxy.ServiceUnavailable.URN(),
				"The service is unreachable. Please try again later."
		}
	}

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

	return codes.Gateway.Proxy.BadGateway.URN(),
		"Unable to connect to the backend service. Please try again in a few moments."
}

func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request()
	startTime := h.Clock.Now()
	var instanceStart, instanceEnd time.Time

	defer func() {
		gatewayDuration := h.Clock.Now().Sub(startTime)
		sess.ResponseWriter().Header().Set("X-Unkey-Gateway-Time", fmt.Sprintf("%dms", gatewayDuration.Milliseconds()))

		if !instanceStart.IsZero() && !instanceEnd.IsZero() {
			instanceDuration := instanceEnd.Sub(instanceStart)
			sess.ResponseWriter().Header().Set("X-Unkey-Instance-Time", fmt.Sprintf("%dms", instanceDuration.Milliseconds()))
		}
	}()

	deploymentID := req.Header.Get("X-Deployment-ID")
	if deploymentID == "" {
		return fault.New("missing deployment ID",
			fault.Code(codes.User.BadRequest.MissingRequiredHeader.URN()),
			fault.Internal("X-Deployment-ID header not set"),
			fault.Public("Bad request"),
		)
	}

	_, err := h.RouterService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return err
	}

	instance, err := h.RouterService.SelectInstance(ctx, deploymentID)
	if err != nil {
		return err
	}

	targetURL, err := url.Parse("http://" + instance.Address)
	if err != nil {
		h.Logger.Error("invalid instance address", "address", instance.Address, "error", err)
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Internal.InvalidConfiguration.URN()),
			fault.Internal("invalid service address"),
			fault.Public("Service configuration error"),
		)
	}

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())

	proxy := &httputil.ReverseProxy{
		Director: func(outReq *http.Request) {
			instanceStart = h.Clock.Now()
			outReq.URL.Scheme = targetURL.Scheme
			outReq.URL.Host = targetURL.Host
			outReq.Host = req.Host

			if outReq.Header == nil {
				outReq.Header = make(http.Header)
			}

			if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
				outReq.Header.Set("X-Forwarded-For", clientIP)
			}
			outReq.Header.Set("X-Forwarded-Host", req.Host)
			outReq.Header.Set("X-Forwarded-Proto", "http")
		},
		Transport: h.Transport,
		ModifyResponse: func(resp *http.Response) error {
			instanceEnd = h.Clock.Now()
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			instanceEnd = h.Clock.Now()

			if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
				ecw.SetError(err)

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

	proxy.ServeHTTP(wrapper, req)

	if err := wrapper.Error(); err != nil {
		urn, message := categorizeProxyError(err)
		return fault.Wrap(err,
			fault.Code(urn),
			fault.Internal(fmt.Sprintf("proxy error forwarding to instance %s", instance.Address)),
			fault.Public(message),
		)
	}

	return nil
}
