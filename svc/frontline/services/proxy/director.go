package proxy

import (
	"net/http"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/zen"
)

// makeSentinelDirector creates a Director function for forwarding to a local sentinel
// The proxyStartTime pointer will be set by the caller when Director is invoked
func (s *service) makeSentinelDirector(sess *zen.Session, deploymentID string, startTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set(HeaderFrontlineID, s.frontlineID)
		req.Header.Set(HeaderRegion, s.region)
		req.Header.Set(HeaderRequestID, sess.RequestID())

		frontlineRoutingTimeMs := s.clock.Now().Sub(startTime).Milliseconds()
		req.Header.Set(HeaderFrontlineTime, strconv.FormatInt(frontlineRoutingTimeMs, 10))

		req.Header.Set(HeaderForwardedProto, "https")

		// We always want to override the deployment id, even if the client send it.
		req.Header.Set(HeaderDeploymentID, deploymentID)
	}
}

// makeRegionDirector creates a Director function for forwarding to a remote region
func (s *service) makeRegionDirector(sess *zen.Session, startTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set(HeaderFrontlineID, s.frontlineID)
		req.Header.Set(HeaderRegion, s.region)
		req.Header.Set(HeaderRequestID, sess.RequestID())

		frontlineRoutingTimeMs := s.clock.Now().Sub(startTime).Milliseconds()
		req.Header.Set(HeaderFrontlineTime, strconv.FormatInt(frontlineRoutingTimeMs, 10))

		// Preserve original Host so we know where to actually route the request
		req.Host = sess.Request().Host
		req.Header.Set("Host", sess.Request().Host)

		// Add parent tracking to trace the forwarding chain, might be useful for debugging
		req.Header.Set(HeaderParentFrontlineID, s.frontlineID)
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

		// Log hop count to wide event for observability
		wide.Set(req.Context(), wide.FieldProxyHops, currentHops)
	}
}
