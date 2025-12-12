package proxy

import (
	"net/http"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/go/pkg/zen"
)

// makeSentinelDirector creates a Director function for forwarding to a local sentinel
// The proxyStartTime pointer will be set by the caller when Director is invoked
func (s *service) makeSentinelDirector(sess *zen.Session, deploymentID string, startTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set(HeaderIngressID, s.ingressID)
		req.Header.Set(HeaderRegion, s.region)
		req.Header.Set(HeaderRequestID, sess.RequestID())

		ingressRoutingTimeMs := s.clock.Now().Sub(startTime).Milliseconds()
		req.Header.Set(HeaderIngressTime, strconv.FormatInt(ingressRoutingTimeMs, 10))

		req.Header.Set(HeaderForwardedProto, "https")

		// We always want to override the deployment id, even if the client send it.
		req.Header.Set(HeaderDeploymentID, deploymentID)
	}
}

// makeRegionDirector creates a Director function for forwarding to a remote region
func (s *service) makeRegionDirector(sess *zen.Session, startTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set(HeaderIngressID, s.ingressID)
		req.Header.Set(HeaderRegion, s.region)
		req.Header.Set(HeaderRequestID, sess.RequestID())

		ingressRoutingTimeMs := s.clock.Now().Sub(startTime).Milliseconds()
		req.Header.Set(HeaderIngressTime, strconv.FormatInt(ingressRoutingTimeMs, 10))

		// Preserve original Host so we know where to actually route the request
		req.Host = sess.Request().Host
		req.Header.Set("Host", sess.Request().Host)

		// Add parent tracking to trace the forwarding chain, might be useful for debugging
		req.Header.Set(HeaderParentIngressID, s.ingressID)
		req.Header.Set(HeaderParentRequestID, sess.RequestID())

		// Parse and increment hop count to prevent infinite loops
		currentHops := 0
		if hopCountStr := req.Header.Get(HeaderIngressHops); hopCountStr != "" {
			if parsed, err := strconv.Atoi(hopCountStr); err == nil {
				currentHops = parsed
			}
		}
		currentHops++
		req.Header.Set(HeaderIngressHops, strconv.Itoa(currentHops))

		if currentHops >= s.maxHops-1 {
			s.logger.Warn("approaching max hops limit",
				"currentHops", currentHops,
				"maxHops", s.maxHops,
				"hostname", req.Host,
			)
		}
	}
}
