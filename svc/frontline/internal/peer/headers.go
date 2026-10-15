package peer

import (
	"net/http"
	"strings"
)

const (
	// HeaderFrontlineMeta carries the signed metadata between frontlines.
	HeaderFrontlineMeta = "X-Unkey-Frontline-Meta"

	headerPrefix         = "X-Unkey-Peer-"
	headerReject         = "X-Unkey-Peer-Reject"
	headerRequestTarget  = "X-Unkey-Peer-Request-Target"
	headerUpgrade        = "X-Unkey-Peer-Upgrade"
	headerUpstreamStatus = "X-Unkey-Peer-Upstream-Status"

	// statusDialed is the private informational status the relay sends once
	// its dial to the pod succeeds.
	statusDialed = 199
)

// Reject reasons sent by the relay before anything reaches the pod.
const (
	rejectUnauthorized    = "unauthorized"
	rejectHops            = "hops"
	rejectInstanceUnknown = "instance_unknown"
	rejectDialFailed      = "dial_failed"
	rejectBadRequest      = "bad_request"
)

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func removePeerHeaders(h http.Header) {
	for name := range h {
		if strings.HasPrefix(name, headerPrefix) {
			delete(h, name)
		}
	}
}

func removeHopHeaders(h http.Header) {
	for _, name := range hopHeaders {
		h.Del(name)
	}
}

func isUpgrade(h http.Header) bool {
	for _, v := range h.Values("Connection") {
		for _, token := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
				return h.Get("Upgrade") != ""
			}
		}
	}
	return false
}
