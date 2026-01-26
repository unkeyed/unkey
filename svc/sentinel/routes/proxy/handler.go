package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/services/router"
)

type Handler struct {
	Logger             logging.Logger
	RouterService      router.Service
	Clock              clock.Clock
	Transport          *http.Transport
	ClickHouse         clickhouse.ClickHouse
	SentinelID         string
	Region             string
	MaxRequestBodySize int64
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request()
	requestID := uid.New("req")
	startTime := h.Clock.Now()
	var instanceStart, instanceEnd time.Time
	var responseStatus int32
	var requestBody, responseBody []byte
	var responseHeaders []string

	deploymentID := req.Header.Get("X-Deployment-Id")
	if deploymentID == "" {
		return fault.New("missing deployment ID",
			fault.Code(codes.User.BadRequest.MissingRequiredHeader.URN()),
			fault.Internal("X-Deployment-Id header not set"),
			fault.Public("Bad request"),
		)
	}

	deployment, err := h.RouterService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return err
	}

	instance, err := h.RouterService.SelectInstance(ctx, deploymentID)
	if err != nil {
		return err
	}

	if req.Body != nil {
		requestBody, err = io.ReadAll(req.Body)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.User.BadRequest.RequestBodyUnreadable.URN()),
				fault.Internal("unable to read request body"),
				fault.Public("The request body could not be read."),
			)
		}
		req.Body = io.NopCloser(bytes.NewReader(requestBody))
	}

	defer func() {
		endTime := h.Clock.Now()
		totalLatency := endTime.Sub(startTime).Milliseconds()
		var instanceLatency int64
		var sentinelLatency int64
		if !instanceStart.IsZero() && !instanceEnd.IsZero() {
			instanceLatency = instanceEnd.Sub(instanceStart).Milliseconds()
			sentinelLatency = totalLatency - instanceLatency
		}

		h.ClickHouse.BufferSentinelRequest(schema.SentinelRequest{
			RequestID:       requestID,
			Time:            startTime.UnixMilli(),
			WorkspaceID:     deployment.WorkspaceID,
			EnvironmentID:   deployment.EnvironmentID,
			ProjectID:       deployment.ProjectID,
			SentinelID:      h.SentinelID,
			DeploymentID:    deploymentID,
			InstanceID:      instance.ID,
			InstanceAddress: instance.Address,
			Region:          h.Region,
			Method:          strings.ToUpper(req.Method),
			Host:            req.Host,
			Path:            req.URL.Path,
			QueryString:     req.URL.RawQuery,
			QueryParams:     req.URL.Query(),
			RequestHeaders:  formatHeaders(req.Header),
			RequestBody:     string(requestBody),
			ResponseStatus:  responseStatus,
			ResponseHeaders: responseHeaders,
			ResponseBody:    string(responseBody),
			UserAgent:       req.UserAgent(),
			IPAddress:       getClientIP(req),
			TotalLatency:    totalLatency,
			InstanceLatency: instanceLatency,
			SentinelLatency: sentinelLatency,
		})
	}()

	targetURL, err := url.Parse("http://" + instance.Address)
	if err != nil {
		h.Logger.Error("invalid instance address", "address", instance.Address, "error", err)
		return fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InvalidConfiguration.URN()),
			fault.Internal("invalid service address"),
			fault.Public("Service configuration error"),
		)
	}

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())
	// nolint:exhaustruct
	proxy := &httputil.ReverseProxy{
		FlushInterval: -1, // flush immediately for streaming
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
			responseStatus = int32(resp.StatusCode)
			responseHeaders = formatHeaders(resp.Header)

			// Capture response body for logging
			if resp.Body != nil {
				responseBody, err = io.ReadAll(resp.Body)
				if err != nil {
					h.Logger.Warn("failed to read response body for logging",
						"error", err.Error(),
						"deploymentID", deploymentID,
						"instanceID", instance.ID,
					)
					responseBody = nil
				}
				resp.Body = io.NopCloser(bytes.NewReader(responseBody))
			}

			sentinelDuration := h.Clock.Now().Sub(startTime)
			resp.Header.Set("X-Unkey-Sentinel-Time", fmt.Sprintf("%dms", sentinelDuration.Milliseconds()))

			if !instanceStart.IsZero() && !instanceEnd.IsZero() {
				instanceDuration := instanceEnd.Sub(instanceStart)
				resp.Header.Set("X-Unkey-Instance-Time", fmt.Sprintf("%dms", instanceDuration.Milliseconds()))
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			instanceEnd = h.Clock.Now()

			sentinelDuration := h.Clock.Now().Sub(startTime)
			sess.ResponseWriter().Header().Set("X-Unkey-Sentinel-Time", fmt.Sprintf("%dms", sentinelDuration.Milliseconds()))

			if !instanceStart.IsZero() && !instanceEnd.IsZero() {
				instanceDuration := instanceEnd.Sub(instanceStart)
				sess.ResponseWriter().Header().Set("X-Unkey-Instance-Time", fmt.Sprintf("%dms", instanceDuration.Milliseconds()))
			}

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

	proxy.ServeHTTP(wrapper, req)

	if err := wrapper.Error(); err != nil {
		urn, message := categorizeProxyError(err)

		// If responseStatus is 0, it means ModifyResponse never ran. No response received from user code.
		// Set a status code based on error type for CH logging.
		if responseStatus == 0 {
			if errors.Is(err, context.Canceled) {
				responseStatus = 499 // Client Closed Request like nginx
			} else if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
				responseStatus = 504 // Gateway Timeout
			} else {
				responseStatus = 502 // Bad Gateway
			}
		}

		return fault.Wrap(err,
			fault.Code(urn),
			fault.Internal(fmt.Sprintf("proxy error forwarding to instance %s", instance.Address)),
			fault.Public(message),
		)
	}

	return nil
}

func categorizeProxyError(err error) (codes.URN, string) {
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Sentinel.Proxy.SentinelTimeout.URN(),
			"The request took too long to process. Please try again later."
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return codes.Sentinel.Proxy.SentinelTimeout.URN(),
				"The request took too long to process. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Sentinel.Proxy.ServiceUnavailable.URN(),
				"The service is temporarily unavailable. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Sentinel.Proxy.BadGateway.URN(),
				"Connection was reset by the backend service. Please try again."
		}

		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Sentinel.Proxy.ServiceUnavailable.URN(),
				"The service is unreachable. Please try again later."
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return codes.Sentinel.Proxy.ServiceUnavailable.URN(),
				"The service could not be found. Please check your configuration."
		}
		if dnsErr.IsTimeout {
			return codes.Sentinel.Proxy.SentinelTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
	}

	return codes.Sentinel.Proxy.BadGateway.URN(),
		"Unable to connect to an instance. Please try again in a few moments."
}

func formatHeaders(headers http.Header) []string {
	result := make([]string, 0, len(headers))
	for key, values := range headers {
		for _, value := range values {
			result = append(result, fmt.Sprintf("%s: %s", key, value))
		}
	}
	return result
}

func getClientIP(req *http.Request) string {
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		return clientIP
	}
	return req.RemoteAddr
}
