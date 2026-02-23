package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
)

type forwardConfig struct {
	targetURL    *url.URL
	startTime    time.Time
	directorFunc func(*http.Request)
	logTarget    string
	transport    http.RoundTripper
}

func (s *service) forward(sess *zen.Session, cfg forwardConfig) error {
	sess.ResponseWriter().Header().Set(HeaderFrontlineID, s.instanceID)
	sess.ResponseWriter().Header().Set(HeaderRegion, s.region)
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	var proxyStartTime time.Time

	defer func() {
		totalTime := s.clock.Now().Sub(cfg.startTime)
		if !proxyStartTime.IsZero() {
			timing.Write(sess.ResponseWriter(), timing.Entry{
				Name:     "frontline",
				Duration: proxyStartTime.Sub(cfg.startTime),
				Attributes: map[string]string{
					"scope": "frontline",
				},
			})
		}
		timing.Write(sess.ResponseWriter(), timing.Entry{
			Name:     "total",
			Duration: totalTime,
			Attributes: map[string]string{
				"scope": "frontline",
			},
		})
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
				timing.Write(sess.ResponseWriter(), timing.Entry{
					Name:     "frontline",
					Duration: proxyStartTime.Sub(cfg.startTime),
					Attributes: map[string]string{
						"scope": "frontline",
					},
				})
			}
			timing.Write(sess.ResponseWriter(), timing.Entry{
				Name:     "total",
				Duration: totalTime,
				Attributes: map[string]string{
					"scope": "frontline",
				},
			})

			if resp.Header.Get("X-Unkey-Error-Source") != "sentinel" {
				return nil
			}

			if sentinelTime := resp.Header.Get(timing.HeaderName); sentinelTime != "" {
				sess.ResponseWriter().Header().Add(timing.HeaderName, sentinelTime)
			}

			// 5xx from sentinel → fault error → frontline observability handles content negotiation
			if resp.StatusCode >= 500 {
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

			// 4xx from sentinel (auth errors, rate limits) → rewrite to HTML if client prefers it,
			// otherwise pass the JSON through untouched.
			if resp.StatusCode >= 400 && wantsHTML(sess.Request()) {
				return rewriteSentinelErrorAsHTML(resp, sess.RequestID(), s.errorPageRenderer)
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			// Capture the error for middleware to handle
			if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
				ecw.SetError(err)

				logger.Warn(fmt.Sprintf("proxy error forwarding to %s", cfg.logTarget),
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

// wantsHTML returns true if the client prefers HTML over JSON based on the Accept header.
func wantsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	for _, part := range strings.Split(accept, ",") {
		mediaType := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		switch mediaType {
		case "text/html":
			return true
		case "application/json", "application/*", "*/*":
			return false
		}
	}

	return false
}

// sentinelError matches the JSON error structure returned by sentinel.
type sentinelError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// rewriteSentinelErrorAsHTML reads the sentinel JSON error response and replaces
// the body with a styled HTML error page. The original status code is preserved.
func rewriteSentinelErrorAsHTML(resp *http.Response, requestID string, renderer errorpage.Renderer) error {
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil // can't read body, let it pass through
	}

	var parsed sentinelError
	if err := json.Unmarshal(body, &parsed); err != nil {
		// Not valid JSON, put the body back unchanged
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	message := parsed.Error.Message
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	title := http.StatusText(resp.StatusCode)
	if title == "" {
		title = "Error"
	}

	var docsURL string
	if parsed.Error.Code != "" {
		if code, parseErr := codes.ParseCode(parsed.Error.Code); parseErr == nil {
			docsURL = code.DocsURL()
		}
	}

	htmlBody, renderErr := renderer.Render(errorpage.Data{
		StatusCode: resp.StatusCode,
		Title:      title,
		Message:    message,
		ErrorCode:  parsed.Error.Code,
		DocsURL:    docsURL,
		RequestID:  requestID,
	})
	if renderErr != nil {
		// Template render failed, put original body back
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	resp.Body = io.NopCloser(bytes.NewReader(htmlBody))
	resp.ContentLength = int64(len(htmlBody))
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(htmlBody)))

	return nil
}
