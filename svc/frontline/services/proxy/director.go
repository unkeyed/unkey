package proxy

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/hoptracing"
	"github.com/unkeyed/unkey/pkg/zen"
)

func (s *service) makeSentinelDirector(sess *zen.Session, trace *hoptracing.Trace, deploymentID string, startTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		durationMs := s.clock.Now().Sub(startTime).Milliseconds()
		trace.AddTiming(hoptracing.HopFrontline, s.region, s.frontlineID, durationMs)

		trace.InjectDownstream(req, deploymentID)
		req.Header.Set(hoptracing.HeaderForwardedProto, "https")
	}
}

func (s *service) makeRegionDirector(sess *zen.Session, trace *hoptracing.Trace, startTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		durationMs := s.clock.Now().Sub(startTime).Milliseconds()
		trace.AddTiming(hoptracing.HopFrontline, s.region, s.frontlineID, durationMs)

		trace.InjectDownstream(req, "")

		req.Host = sess.Request().Host
		req.Header.Set("Host", sess.Request().Host)

		if trace.HopCount >= s.maxHops-1 {
			s.logger.Warn("approaching max hops limit",
				"currentHops", trace.HopCount,
				"maxHops", s.maxHops,
				"hostname", req.Host,
			)
		}
	}
}
