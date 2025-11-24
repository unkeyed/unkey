package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type forwardConfig struct {
	targetURL    *url.URL
	startTime    time.Time
	directorFunc func(*http.Request)
	logTarget    string
}

func (s *service) forward(sess *zen.Session, cfg forwardConfig) error {
	sess.ResponseWriter().Header().Set(HeaderIngressID, s.ingressID)
	sess.ResponseWriter().Header().Set(HeaderRegion, s.region)
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	var proxyStartTime time.Time

	defer func() {
		totalTime := s.clock.Now().Sub(cfg.startTime)
		if !proxyStartTime.IsZero() {
			sess.ResponseWriter().Header().Set("X-Unkey-Ingress-Time", fmt.Sprintf("%dms", proxyStartTime.Sub(cfg.startTime).Milliseconds()))
		}
		sess.ResponseWriter().Header().Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))
	}()

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())

	proxy := &httputil.ReverseProxy{
		Transport: s.transport,
		Director: func(req *http.Request) {
			proxyStartTime = s.clock.Now()

			req.URL.Scheme = cfg.targetURL.Scheme
			req.URL.Host = cfg.targetURL.Host

			cfg.directorFunc(req)
		},
		ModifyResponse: func(resp *http.Response) error {
			totalTime := s.clock.Now().Sub(cfg.startTime)
			if !proxyStartTime.IsZero() {
				resp.Header.Set("X-Unkey-Ingress-Time", fmt.Sprintf("%dms", proxyStartTime.Sub(cfg.startTime).Milliseconds()))
			}
			resp.Header.Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))

			if resp.StatusCode >= 500 && resp.Header.Get("X-Unkey-Error-Source") == "gateway" {
				if gatewayTime := resp.Header.Get("X-Unkey-Gateway-Time"); gatewayTime != "" {
					sess.ResponseWriter().Header().Set("X-Unkey-Gateway-Time", gatewayTime)
				}
				if instanceTime := resp.Header.Get("X-Unkey-Instance-Time"); instanceTime != "" {
					sess.ResponseWriter().Header().Set("X-Unkey-Instance-Time", instanceTime)
				}

				urn := codes.Ingress.Proxy.BadGateway.URN()
				if resp.StatusCode == 503 {
					urn = codes.Ingress.Proxy.ServiceUnavailable.URN()
				} else if resp.StatusCode == 504 {
					urn = codes.Ingress.Proxy.GatewayTimeout.URN()
				}

				return fault.New(
					fmt.Sprintf("gateway returned %d", resp.StatusCode),
					fault.Code(urn),
					fault.Public(http.StatusText(resp.StatusCode)),
				)
			}

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
