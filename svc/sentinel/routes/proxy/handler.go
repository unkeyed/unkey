package handler

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
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
	Transport          http.RoundTripper
	SentinelID         string
	Region             string
	MaxRequestBodySize int64
	Engine             engine.Evaluator
	RetryBudget        *retryBudget
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
			principalJSON, serErr := engine.SerializePrincipal(result.Principal)
			if serErr != nil {
				logger.Error("failed to serialize principal", "error", serErr)
			} else {
				req.Header.Set(engine.PrincipalHeader, principalJSON)
			}
		}
	}

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
	}

	// First attempt
	instance, err := h.RouterService.SelectInstance(ctx, deploymentID)
	if err != nil {
		return err
	}

	attempt := &proxyAttempt{
		sess:         sess,
		req:          req,
		tracking:     tracking,
		requestID:    requestID,
		deploymentID: deploymentID,
		deployment:   deployment,
		instance:     instance,
		requestBody:  requestBody,
	}

	proxyErr := h.proxyToInstance(ctx, attempt)

	// Retry on a (potentially different) instance if the error is retryable.
	// The first instance's inflight count is already released, so P2C will
	// naturally prefer a different instance if one is available.
	if proxyErr != nil && isRetryableError(proxyErr) && h.RetryBudget.Withdraw() {
		retryInstance, selectErr := h.RouterService.SelectInstance(ctx, deploymentID)
		if selectErr != nil {
			// Can't get a retry instance, return original error.
			return proxyErr
		}

		attempt.instance = retryInstance
		proxyErr = h.proxyToInstance(ctx, attempt)
	} else if proxyErr == nil {
		// Non-retried success: deposit a token.
		h.RetryBudget.Deposit()
	}

	return proxyErr
}

// proxyAttempt bundles the state needed for a single proxy attempt.
type proxyAttempt struct {
	sess         *zen.Session
	req          *http.Request
	tracking     *SentinelRequestTracking
	requestID    string
	deploymentID string
	deployment   db.Deployment
	instance     db.Instance
	requestBody  []byte
}

// proxyToInstance performs the reverse proxy to a single instance. It handles
// inflight tracking (release on return), body rewinding, tracking context
// updates, and timing headers.
func (h *Handler) proxyToInstance(ctx context.Context, a *proxyAttempt) error {
	instance := a.instance
	defer h.RouterService.ReleaseInstance(instance.ID)

	// Rewind body for this attempt.
	if a.requestBody != nil {
		a.req.Body = io.NopCloser(bytes.NewReader(a.requestBody))
		a.req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(a.requestBody)), nil
		}
	}

	// Update tracking context for this attempt.
	if a.tracking != nil {
		a.tracking.RequestID = a.requestID
		a.tracking.DeploymentID = a.deploymentID
		a.tracking.Deployment = &DeploymentInfo{
			WorkspaceID:   a.deployment.WorkspaceID,
			EnvironmentID: a.deployment.EnvironmentID,
			ProjectID:     a.deployment.ProjectID,
		}
		a.tracking.Instance = &InstanceInfo{
			ID:      instance.ID,
			Address: instance.Address,
		}
		a.tracking.RequestBody = a.requestBody
		// Reset timing for this attempt.
		a.tracking.InstanceStart = h.Clock.Now()
		a.tracking.InstanceEnd = a.tracking.InstanceStart
		a.tracking.ResponseStatus = 0
		a.tracking.ResponseHeaders = nil
		a.tracking.ResponseBody = nil
	}

	targetURL, err := url.Parse("http://" + instance.Address)
	if err != nil {
		logger.Error("invalid instance address", "address", instance.Address, "error", err)
		return fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InvalidConfiguration.URN()),
			fault.Internal("invalid service address"),
			fault.Public("Service configuration error"),
		)
	}

	wrapper := zen.NewErrorCapturingWriter(a.sess.ResponseWriter())
	// nolint:exhaustruct
	proxy := &httputil.ReverseProxy{
		FlushInterval: -1, // flush immediately for streaming
		Director: func(outReq *http.Request) {
			if a.tracking != nil {
				a.tracking.InstanceStart = h.Clock.Now()
			}

			outReq.URL.Scheme = targetURL.Scheme
			outReq.URL.Host = targetURL.Host
			outReq.Host = a.req.Host

			if outReq.Header == nil {
				outReq.Header = make(http.Header)
			}

			if clientIP, _, err := net.SplitHostPort(a.req.RemoteAddr); err == nil {
				outReq.Header.Set("X-Forwarded-For", clientIP)
			}
			outReq.Header.Set("X-Forwarded-Host", a.req.Host)
			outReq.Header.Set("X-Forwarded-Proto", "http")
		},
		Transport: h.Transport,
		ModifyResponse: func(resp *http.Response) error {
			if a.tracking != nil {
				a.tracking.InstanceEnd = h.Clock.Now()
				a.tracking.ResponseStatus = int32(resp.StatusCode)
				a.tracking.ResponseHeaders = resp.Header

				// Record upstream metrics
				statusClass := upstreamStatusClass(resp.StatusCode)
				upstreamResponseTotal.WithLabelValues(statusClass).Inc()
				if !a.tracking.InstanceStart.IsZero() {
					upstreamDuration.WithLabelValues(statusClass).Observe(a.tracking.InstanceEnd.Sub(a.tracking.InstanceStart).Seconds())
				}

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
					a.tracking.ResponseBody = responseBody
					resp.Body = io.NopCloser(bytes.NewReader(responseBody))
				}
			}

			if a.tracking != nil {
				sentinelDuration := h.Clock.Now().Sub(a.tracking.StartTime)
				timing.Write(a.sess.ResponseWriter(), timing.Entry{
					Name:     "sentinel",
					Duration: sentinelDuration,
					Attributes: map[string]string{
						"scope": "sentinel",
					},
				})

				if !a.tracking.InstanceStart.IsZero() && !a.tracking.InstanceEnd.IsZero() {
					instanceDuration := a.tracking.InstanceEnd.Sub(a.tracking.InstanceStart)
					timing.Write(a.sess.ResponseWriter(), timing.Entry{
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
			if a.tracking != nil {
				a.tracking.InstanceEnd = h.Clock.Now()

				sentinelDuration := h.Clock.Now().Sub(a.tracking.StartTime)
				timing.Write(a.sess.ResponseWriter(), timing.Entry{
					Name:     "sentinel",
					Duration: sentinelDuration,
					Attributes: map[string]string{
						"scope": "sentinel",
					},
				})

				if !a.tracking.InstanceStart.IsZero() && !a.tracking.InstanceEnd.IsZero() {
					instanceDuration := a.tracking.InstanceEnd.Sub(a.tracking.InstanceStart)
					timing.Write(a.sess.ResponseWriter(), timing.Entry{
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

	proxy.ServeHTTP(wrapper, a.req)
	return wrapper.Error()
}
