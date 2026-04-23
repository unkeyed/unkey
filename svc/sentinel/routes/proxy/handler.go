package handler

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/engine"
	"github.com/unkeyed/unkey/svc/sentinel/services/router"
)

type Handler struct {
	RouterService      router.Service
	Clock              clock.Clock
	Transports         *TransportRegistry
	SentinelID         string
	Region             string
	MaxRequestBodySize int64
	Engine             engine.Evaluator
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request().WithContext(ctx)

	tracking, ok := SentinelTrackingFromContext(ctx)
	if !ok {
		logger.Warn("no sentinel tracking context found")
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

	deployment, err := h.RouterService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return err
	}

	instance, err := h.RouterService.SelectInstance(ctx, deploymentID)
	if err != nil {
		return err
	}

	// Always strip incoming X-Unkey-Principal header to prevent spoofing
	req.Header.Del(engine.PrincipalHeader)

	// Evaluate sentinel middleware policies (parsed + cached by the router service)
	mw, err := h.RouterService.GetPolicies(ctx, deployment)
	if err != nil {
		return err
	}
	if mw != nil && h.Engine != nil {
		result, evalErr := h.Engine.Evaluate(ctx, sess, req, mw)
		if evalErr != nil {
			return evalErr
		}

		if result.Principal != nil {
			principalJSON, serErr := result.Principal.Marshal()
			if serErr != nil {
				logger.Error("failed to serialize principal", "error", serErr)
			} else {
				req.Header.Set(engine.PrincipalHeader, principalJSON)
			}
		}
	}

	streaming := strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc") ||
		strings.HasPrefix(req.Header.Get("Content-Type"), "application/connect+")

	// Capture the request body for logging.
	// Streaming: TeeReader passes bytes through to the upstream while capturing a copy.
	// Non-streaming: read the full body upfront so we can fail fast on unreadable bodies.
	var requestBuf bytes.Buffer
	var requestBody []byte
	if req.Body != nil && streaming {
		req.Body = io.NopCloser(io.TeeReader(req.Body, &zen.LimitedWriter{W: &requestBuf, N: zen.MaxBodyCapture}))
	} else if req.Body != nil {
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

	transport := h.Transports.Get(deployment.UpstreamProtocol)

	targetURL, err := url.Parse("http://" + instance.Address)
	if err != nil {
		logger.Error("invalid instance address", "address", instance.Address, "error", err)
		return fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InvalidConfiguration.URN()),
			fault.Internal("invalid service address"),
			fault.Public("Service configuration error"),
		)
	}

	// Buffer to capture streaming response body via TeeReader.
	// Bytes are written here as they stream through to the client,
	// so the full body is available for logging after proxy.ServeHTTP returns.
	var responseBuf bytes.Buffer

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
		Transport: transport,
		ModifyResponse: func(resp *http.Response) error {
			if tracking != nil {
				tracking.ResponseStatus = int32(resp.StatusCode)
				tracking.ResponseHeaders = resp.Header

				// Record time-to-first-byte metrics (InstanceEnd is set after the
				// full stream completes in the post-proxy block below).
				ttfb := h.Clock.Now()
				statusClass := upstreamStatusClass(resp.StatusCode)
				upstreamResponseTotal.WithLabelValues(statusClass).Inc()
				if !tracking.InstanceStart.IsZero() {
					upstreamDuration.WithLabelValues(statusClass).Observe(ttfb.Sub(tracking.InstanceStart).Seconds())
				}

				// Capture response body for logging.
				// Always use TeeReader — bytes flow through to the client while
				// accumulating in responseBuf (capped at MaxBodyCapture).
				// This avoids buffering the entire body (which blocks streaming)
				// and removes the need to detect streaming via Content-Type.
				if resp.Body != nil {
					responseBuf.Reset()
					resp.Body = io.NopCloser(io.TeeReader(resp.Body, &zen.LimitedWriter{W: &responseBuf, N: zen.MaxBodyCapture}))
				}
			}

			if tracking != nil {
				sentinelDuration := h.Clock.Now().Sub(tracking.StartTime)
				timing.Write(sess.ResponseWriter(), timing.Entry{
					Name:     "sentinel",
					Duration: sentinelDuration,
					Attributes: map[string]string{
						"scope": "sentinel",
					},
				})

				if !tracking.InstanceStart.IsZero() && !tracking.InstanceEnd.IsZero() {
					instanceDuration := tracking.InstanceEnd.Sub(tracking.InstanceStart)
					timing.Write(sess.ResponseWriter(), timing.Entry{
						Name:     "instance",
						Duration: instanceDuration,
						Attributes: map[string]string{
							"scope": "sentinel",
						},
					})
				}
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if tracking != nil {
				sentinelDuration := h.Clock.Now().Sub(tracking.StartTime)
				timing.Write(sess.ResponseWriter(), timing.Entry{
					Name:     "sentinel",
					Duration: sentinelDuration,
					Attributes: map[string]string{
						"scope": "sentinel",
					},
				})

				if !tracking.InstanceStart.IsZero() && !tracking.InstanceEnd.IsZero() {
					instanceDuration := tracking.InstanceEnd.Sub(tracking.InstanceStart)
					timing.Write(sess.ResponseWriter(), timing.Entry{
						Name:     "instance",
						Duration: instanceDuration,
						Attributes: map[string]string{
							"scope": "sentinel",
						},
					})
				}
			}

			if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
				ecw.SetError(err)
			}
		},
	}

	serveWithAbortRecovery(proxy, wrapper, req)

	// Mark the true end of the instance interaction (includes full stream duration).
	if tracking != nil {
		tracking.InstanceEnd = h.Clock.Now()
	}

	// TeeReaders have now accumulated the full bodies for any streaming direction.
	if tracking != nil {
		if streaming && requestBuf.Len() > 0 {
			tracking.RequestBody = requestBuf.Bytes()
		}
		if responseBuf.Len() > 0 {
			tracking.ResponseBody = responseBuf.Bytes()
		}
	}

	// Feed the captured response body back into the session so zen middleware
	// (WithLogging, WithMetrics) can log it — the proxy writes directly to
	// the ResponseWriter, bypassing Session.send().
	if tracking != nil && len(tracking.ResponseBody) > 0 {
		sess.SetResponseBody(tracking.ResponseBody)
	}

	return wrapper.Error()
}

// serveWithAbortRecovery calls h.ServeHTTP and swallows http.ErrAbortHandler panics.
// httputil.ReverseProxy panics with that sentinel when the response body copy fails
// after headers have been flushed (typically the client disconnected mid-stream).
// There is no recovery path at that point — the panic is the library's signal that
// the handler is aborting. Swallowing it keeps the global panic middleware strict
// for real bugs; other panic values are re-raised unchanged.
func serveWithAbortRecovery(h http.Handler, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			if rec != http.ErrAbortHandler {
				panic(rec)
			}
			proxyAbortedTotal.Inc()
		}
	}()
	h.ServeHTTP(w, r)
}
