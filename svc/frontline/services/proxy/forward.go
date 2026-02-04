package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/zen"
)

type forwardConfig struct {
	ctx          context.Context
	targetURL    *url.URL
	startTime    time.Time
	directorFunc func(*http.Request)
	logTarget    string
	transport    http.RoundTripper
}

func (s *service) forward(sess *zen.Session, cfg forwardConfig) error {
	// Log upstream URL early so it's available in wide event even if errors occur
	wide.Set(cfg.ctx, wide.FieldUpstreamURL, cfg.targetURL.String())
	sess.ResponseWriter().Header().Set(HeaderFrontlineID, s.frontlineID)
	sess.ResponseWriter().Header().Set(HeaderRegion, s.region)
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	var proxyStartTime time.Time

	defer func() {
		totalTime := s.clock.Now().Sub(cfg.startTime)
		if !proxyStartTime.IsZero() {
			sess.ResponseWriter().Header().Set("X-Unkey-Frontline-Time", fmt.Sprintf("%dms", proxyStartTime.Sub(cfg.startTime).Milliseconds()))
		}
		sess.ResponseWriter().Header().Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))
	}()

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())
	// nolint:exhaustruct
	proxy := &httputil.ReverseProxy{
		Transport:     cfg.transport,
		FlushInterval: -1, // flush immediately for streaming
		Director: func(req *http.Request) {
			proxyStartTime = s.clock.Now()

			req.URL.Scheme = cfg.targetURL.Scheme
			req.URL.Host = cfg.targetURL.Host

			cfg.directorFunc(req)
		},
		ModifyResponse: func(resp *http.Response) error {
			totalTime := s.clock.Now().Sub(cfg.startTime)
			if !proxyStartTime.IsZero() {
				resp.Header.Set("X-Unkey-Frontline-Time", fmt.Sprintf("%dms", proxyStartTime.Sub(cfg.startTime).Milliseconds()))
			}
			resp.Header.Set("X-Unkey-Total-Time", fmt.Sprintf("%dms", totalTime.Milliseconds()))

			if resp.StatusCode >= 500 && resp.Header.Get("X-Unkey-Error-Source") == "sentinel" {
				if sentinelTime := resp.Header.Get("X-Unkey-Sentinel-Time"); sentinelTime != "" {
					sess.ResponseWriter().Header().Set("X-Unkey-Sentinel-Time", sentinelTime)
				}
				if instanceTime := resp.Header.Get("X-Unkey-Instance-Time"); instanceTime != "" {
					sess.ResponseWriter().Header().Set("X-Unkey-Instance-Time", instanceTime)
				}

				urn := codes.Frontline.Proxy.BadGateway.URN()
				switch resp.StatusCode {
				case http.StatusServiceUnavailable:
					urn = codes.Frontline.Proxy.ServiceUnavailable.URN()
				case http.StatusGatewayTimeout:
					urn = codes.Frontline.Proxy.GatewayTimeout.URN()
				case http.StatusBadGateway:
					urn = codes.Frontline.Proxy.BadGateway.URN()
				}

				return fault.New(
					fmt.Sprintf("sentinel returned %d", resp.StatusCode),
					fault.Code(urn),
					fault.Public(http.StatusText(resp.StatusCode)),
				)
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			// Capture the error for middleware to handle
			// Wide event logging is handled by middleware - no separate log needed here
			if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
				ecw.SetError(err)
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
