package proxy

import (
	"net/http"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/go/pkg/zen"
)

// makeGatewayDirector creates a Director function for forwarding to a local gateway
func (s *service) makeGatewayDirector(sess *zen.Session, deploymentID string, startTime time.Time, proxyStartTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		// Add metadata headers TO DOWNSTREAM SERVICE (gateway)
		// These tell the gateway which ingress forwarded the request
		req.Header.Set(HeaderIngressID, s.ingressID)
		req.Header.Set(HeaderRegion, s.region)
		req.Header.Set(HeaderRequestID, sess.RequestID())

		// Add timing to track latency added by this ingress (routing overhead)
		ingressRoutingTimeMs := proxyStartTime.Sub(startTime).Milliseconds()
		req.Header.Set(HeaderIngressTime, strconv.FormatInt(ingressRoutingTimeMs, 10))

		// Add standard proxy headers for local gateway
		req.Header.Set(HeaderForwardedProto, "https")

		// Add deployment ID so gateway knows which deployment to route to
		req.Header.Set(HeaderDeploymentID, deploymentID)
	}
}

// makeNLBDirector creates a Director function for forwarding to a remote NLB
func (s *service) makeNLBDirector(sess *zen.Session, startTime time.Time, proxyStartTime time.Time) func(*http.Request) {
	return func(req *http.Request) {
		// Add metadata headers TO DOWNSTREAM SERVICE (remote ingress)
		// These tell the remote ingress which ingress forwarded the request
		req.Header.Set(HeaderIngressID, s.ingressID)
		req.Header.Set(HeaderRegion, s.region)
		req.Header.Set(HeaderRequestID, sess.RequestID())

		// Add timing to track latency added by this ingress (routing overhead)
		ingressRoutingTimeMs := proxyStartTime.Sub(startTime).Milliseconds()
		req.Header.Set(HeaderIngressTime, strconv.FormatInt(ingressRoutingTimeMs, 10))

		// Remote ingress - preserve original Host for TLS termination and routing
		req.Host = sess.Request().Host
		req.Header.Set("Host", sess.Request().Host)

		// Add parent tracking to trace the forwarding chain
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

		// Log warning if approaching max hops
		if currentHops >= s.maxHops-1 {
			s.logger.Warn("approaching max hops limit",
				"currentHops", currentHops,
				"maxHops", s.maxHops,
				"hostname", req.Host,
			)
		}
	}
}
