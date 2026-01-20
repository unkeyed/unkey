package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/pkg/hoptracing"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/services/resilience"
	"go.opentelemetry.io/otel/attribute"
)

type forwardConfig struct {
	targetURL    *url.URL
	startTime    time.Time
	trace        *hoptracing.Trace
	directorFunc func(*http.Request)
	logTarget    string
	transport    http.RoundTripper
	targetRegion string
}

func (s *service) forward(sess *zen.Session, cfg forwardConfig) error {
	ctx, span := tracing.Start(sess.Request().Context(), "proxy.forward")
	defer span.End()
	span.SetAttributes(
		attribute.String("target", cfg.logTarget),
		attribute.String("target_url", cfg.targetURL.String()),
		attribute.String("target_region", cfg.targetRegion),
	)

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())

	//nolint:exhaustruct
	proxy := &httputil.ReverseProxy{
		Transport:     cfg.transport,
		FlushInterval: -1,
		Director: func(req *http.Request) {
			req.URL.Scheme = cfg.targetURL.Scheme
			req.URL.Host = cfg.targetURL.Host

			if cfg.targetRegion != "" {
				ctx := resilience.WithKey(req.Context(), cfg.targetRegion)
				*req = *req.WithContext(ctx)
			}

			cfg.directorFunc(req)
		},
		ModifyResponse: func(resp *http.Response) error {
			hoptracing.MergeDownstreamTiming(cfg.trace, resp.Header.Get(hoptracing.HeaderTiming))
			hoptracing.MergeDownstreamRoute(cfg.trace, resp.Header.Get(hoptracing.HeaderRoute))

			if pf, isError := ClassifyUpstreamResponse(resp); isError {
				totalMs := s.clock.Now().Sub(cfg.startTime).Milliseconds()
				cfg.trace.InjectResponse(sess.ResponseWriter().Header(), totalMs)
				return pf.ToFault(fmt.Sprintf("%s returned %d", cfg.logTarget, resp.StatusCode))
			}

			totalMs := s.clock.Now().Sub(cfg.startTime).Milliseconds()
			cfg.trace.InjectResponse(resp.Header, totalMs)

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
				ecw.SetError(err)

				s.logger.Warn(fmt.Sprintf("proxy error forwarding to %s", cfg.logTarget),
					"error", err.Error(),
					"target", cfg.targetURL.String(),
					"hostname", r.Host,
					"traceId", cfg.trace.TraceID,
				)
			}

			totalMs := s.clock.Now().Sub(cfg.startTime).Milliseconds()
			cfg.trace.InjectResponse(sess.ResponseWriter().Header(), totalMs)
		},
	}

	proxy.ServeHTTP(wrapper, sess.Request().WithContext(ctx))

	if err := wrapper.Error(); err != nil {
		tracing.RecordError(span, err)
		pf := ClassifyNetworkError(err)
		return pf.ToFault(fmt.Sprintf("proxy error forwarding to %s %s", cfg.logTarget, cfg.targetURL.String()))
	}

	return nil
}
