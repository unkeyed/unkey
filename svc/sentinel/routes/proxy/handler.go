package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/wide"
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

	tracking, ok := SentinelTrackingFromContext(ctx)
	if !ok {
		h.Logger.Warn("no sentinel tracking context found")
	}

	requestID := uid.New("req")

	deploymentID := req.Header.Get("X-Deployment-Id")
	if deploymentID == "" {
		return fault.New("missing deployment ID",
			fault.Code(codes.User.BadRequest.MissingRequiredHeader.URN()),
			fault.Internal("X-Deployment-Id header not set"),
			fault.Public("Bad request"),
		)
	}

	wide.Set(ctx, wide.FieldDeploymentID, deploymentID)

	deployment, err := h.RouterService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return err
	}

	instance, err := h.RouterService.SelectInstance(ctx, deploymentID)
	if err != nil {
		return err
	}

	wide.Set(ctx, wide.FieldInstanceID, instance.ID)
	wide.Set(ctx, wide.FieldUpstreamHost, instance.Address)

	var requestBody []byte
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
		wide.Set(ctx, wide.FieldRequestBodySize, len(requestBody))
	}

	// Populate tracking context
	if tracking != nil {
		tracking.RequestID = requestID
		tracking.DeploymentID = deploymentID
		tracking.Deployment = &DeploymentInfo{
			WorkspaceID:   deployment.WorkspaceID,
			EnvironmentID: deployment.EnvironmentID,
			ProjectID:     deployment.ProjectID,
		}
		tracking.Instance = &InstanceInfo{
			ID:      instance.ID,
			Address: instance.Address,
		}
		tracking.RequestBody = requestBody
	}

	targetURL, err := url.Parse("http://" + instance.Address)
	if err != nil {
		h.Logger.Error("invalid instance address", "address", instance.Address, "error", err)
		return fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InvalidConfiguration.URN()),
			fault.Internal("invalid service address"),
			fault.Public("Service configuration error"),
		)
	}

	wide.Set(ctx, wide.FieldUpstreamURL, targetURL.String())

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())
	// nolint:exhaustruct
	proxy := &httputil.ReverseProxy{
		FlushInterval: -1, // flush immediately for streaming
		Director: func(outReq *http.Request) {
			if tracking != nil {
				tracking.InstanceStart = h.Clock.Now()
			}

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
			if tracking != nil {
				tracking.InstanceEnd = h.Clock.Now()
				tracking.ResponseStatus = int32(resp.StatusCode)
				tracking.ResponseHeaders = resp.Header

				// Capture response body for logging
				if resp.Body != nil {
					responseBody, err := io.ReadAll(resp.Body)
					if err != nil {
						return fault.Wrap(err,
							fault.Code(codes.Sentinel.Proxy.BadGateway.URN()),
							fault.Internal("failed to read response body for logging"),
							fault.Public("Failed to process backend response"),
						)
					}
					tracking.ResponseBody = responseBody
					resp.Body = io.NopCloser(bytes.NewReader(responseBody))
				}
			}

			if tracking != nil {
				sentinelDuration := h.Clock.Now().Sub(tracking.StartTime)
				resp.Header.Set("X-Unkey-Sentinel-Time", fmt.Sprintf("%dms", sentinelDuration.Milliseconds()))

				if !tracking.InstanceStart.IsZero() && !tracking.InstanceEnd.IsZero() {
					instanceDuration := tracking.InstanceEnd.Sub(tracking.InstanceStart)
					resp.Header.Set("X-Unkey-Instance-Time", fmt.Sprintf("%dms", instanceDuration.Milliseconds()))
				}
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if tracking != nil {
				tracking.InstanceEnd = h.Clock.Now()

				sentinelDuration := h.Clock.Now().Sub(tracking.StartTime)
				sess.ResponseWriter().Header().Set("X-Unkey-Sentinel-Time", fmt.Sprintf("%dms", sentinelDuration.Milliseconds()))

				if !tracking.InstanceStart.IsZero() && !tracking.InstanceEnd.IsZero() {
					instanceDuration := tracking.InstanceEnd.Sub(tracking.InstanceStart)
					sess.ResponseWriter().Header().Set("X-Unkey-Instance-Time", fmt.Sprintf("%dms", instanceDuration.Milliseconds()))
				}
			}

			if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
				ecw.SetError(err)
			}
		},
	}

	proxy.ServeHTTP(wrapper, req)
	return wrapper.Error()
}
