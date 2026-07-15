package proxy

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/zen"
)

// makeInstanceDirector creates the Director used when proxying directly to a
// deployment instance. The handler has already evaluated policies and (when
// keyauth produced one) set the X-Unkey-Principal header on the inbound
// request — we just forward it along with standard X-Forwarded-* headers.
func (s *service) makeInstanceDirector(sess *zen.Session, startTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set(HeaderFrontlineID, s.instanceID)
		req.Header.Set(HeaderRegion, fmt.Sprintf("%s::%s", s.platform, s.region))
		req.Header.Set(HeaderRequestID, sess.RequestID())

		frontlineRoutingTime := s.clock.Now().Sub(startTime)
		timing.Write(sess.ResponseWriter(), timing.Entry{
			Name:     "frontline_routing",
			Duration: frontlineRoutingTime,
			Attributes: map[string]string{
				"scope": "frontline",
			},
		})

		// Preserve original Host so the upstream sees what the client asked for.
		req.Host = sess.Request().Host

		if clientIP, _, err := net.SplitHostPort(sess.Request().RemoteAddr); err == nil {
			req.Header.Set("X-Forwarded-For", clientIP)
		} else if loc := sess.Location(); loc != "" {
			req.Header.Set("X-Forwarded-For", loc)
		}
		req.Header.Set("X-Forwarded-Host", sess.Request().Host)
		req.Header.Set("X-Forwarded-Proto", "https")
	}
}

// makeRegionDirector creates a Director function for forwarding to a remote region
func (s *service) makeRegionDirector(sess *zen.Session, startTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set(HeaderFrontlineID, s.instanceID)
		req.Header.Set(HeaderRegion, fmt.Sprintf("%s::%s", s.platform, s.region))
		req.Header.Set(HeaderRequestID, sess.RequestID())

		frontlineRoutingTime := s.clock.Now().Sub(startTime)
		timing.Write(sess.ResponseWriter(), timing.Entry{
			Name:     "frontline_routing",
			Duration: frontlineRoutingTime,
			Attributes: map[string]string{
				"scope": "frontline",
			},
		})

		// Preserve original Host so we know where to actually route the request
		req.Host = sess.Request().Host
		req.Header.Set("Host", sess.Request().Host)

		// Add parent tracking to trace the forwarding chain, might be useful for debugging
		req.Header.Set(HeaderParentFrontlineID, s.instanceID)
		req.Header.Set(HeaderParentRequestID, sess.RequestID())

		// Parse and increment hop count to prevent infinite loops
		currentHops := 0
		if hopCountStr := req.Header.Get(HeaderFrontlineHops); hopCountStr != "" {
			if parsed, err := strconv.Atoi(hopCountStr); err == nil {
				currentHops = parsed
			}
		}
		currentHops++
		req.Header.Set(HeaderFrontlineHops, strconv.Itoa(currentHops))

		if currentHops >= s.maxHops-1 {
			logger.Warn("approaching max hops limit",
				"currentHops", currentHops,
				"maxHops", s.maxHops,
				"hostname", req.Host,
			)
		}
	}
}
