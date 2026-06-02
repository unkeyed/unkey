package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/unkeyed/unkey/pkg/codes"
)

// IsDialError reports whether err is a dial-phase failure — i.e. the proxy
// never established a TCP connection to the upstream. In that case the
// request body has not been read and the request can safely be replayed
// against a different instance.
//
// Mid-stream failures (ECONNRESET on a connection that was already writing,
// response timeouts, context cancellation) are NOT dial errors: the upstream
// may already have processed the request, so a retry would risk double-execute
// on non-idempotent endpoints.
func IsDialError(err error) bool {
	if err == nil {
		return false
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) && netErr.Op == "dial" {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return false
}

// isDialOp reports whether err originated from a "dial" net.OpError. Used
// to distinguish dial-phase timeouts (dial_timeout) from after-dial
// response timeouts (upstream_response_timeout). Wraps are unwrapped.
func isDialOp(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) && netErr.Op == "dial" {
		return true
	}
	return false
}

// categorizeProxyError maps a raw upstream / dial error into a stable
// codes.URN plus a public-facing message. The URN drives status code
// selection in middleware; the message is what the client sees in the
// rendered error page or JSON body.
//
// destination is the kind of upstream — destinationInstance (customer
// pod) or destinationFrontline (peer frontline) — and picks between
// the two taxonomies:
//
//   - Instance failures use upstream_* URNs (OutcomeUpstreamProblem).
//   - Peer-frontline failures use peer_frontline_* URNs
//     (OutcomeFrontlineFault).
//
// Callers should preserve any pre-existing fault code via fault.GetCode
// before calling this — it has no way to know whether err was already
// classified upstream.
func categorizeProxyError(err error, destination string) (codes.URN, string) {
	// Cross-destination cases that don't depend on the upstream kind.
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		// The outer request deadline (RequestTimeout) fired while waiting.
		// On the instance path this means the customer app was slower than
		// our timeout — an upstream problem, not a frontline fault, so it
		// must not page on-call. UpstreamResponseTimeout buckets as
		// OutcomeUpstreamProblem. On the peer-frontline path it's a hop we
		// own, so GatewayDeadlineExceeded (OutcomeFrontlineFault) is right.
		if destination == destinationInstance {
			return codes.Frontline.Proxy.UpstreamResponseTimeout.URN(),
				fmt.Sprintf("The %s did not respond in time. Please try again later.", destination)
		}
		return codes.Frontline.Proxy.GatewayDeadlineExceeded.URN(),
			fmt.Sprintf("The %s did not respond before our deadline. Please try again later.", destination)
	}

	if destination == destinationFrontline {
		return classifyPeerFrontlineError(err)
	}
	return classifyInstanceError(err)
}

// classifyInstanceError maps a wire-level error talking to a customer
// instance to its upstream_* URN. Every URN produced here maps to
// OutcomeUpstreamProblem in the metrics layer except
// ProxyErrorUnclassified, which is the frontline_fault fallback so we get
// paged on errors we don't recognise yet.
func classifyInstanceError(err error) (codes.URN, string) {
	const target = "instance"

	// DNS shouldn't fire on this path — we dial customer instances by
	// IP. If a DNS error reaches here, the dialer was handed a
	// hostname when it should have had an IP, which is a frontline
	// bug, not anything the customer can fix. Route to
	// ProxyErrorUnclassified (frontline_fault outcome) so on-call gets
	// paged. The branch also short-circuits before the generic
	// timeout branch below, which would otherwise claim DNS timeouts
	// (*net.DNSError satisfies os.IsTimeout) and silently misattribute
	// them as response timeouts.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return codes.Frontline.Proxy.ProxyErrorUnclassified.URN(),
			fmt.Sprintf("Failed to connect to the %s. Please try again or contact support at support@unkey.com.", target)
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			if netErr.Op == "dial" {
				return codes.Frontline.Proxy.DialTimeout.URN(),
					fmt.Sprintf("The %s did not accept the connection in time. Please try again later.", target)
			}
			return codes.Frontline.Proxy.UpstreamResponseTimeout.URN(),
				fmt.Sprintf("The %s did not respond in time. Please try again later.", target)
		}

		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Frontline.Proxy.UpstreamConnectionRefused.URN(),
				fmt.Sprintf("The %s refused the connection. Please try again in a few seconds.", target)
		}
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Frontline.Proxy.UpstreamConnectionReset.URN(),
				fmt.Sprintf("The %s reset the connection. Please try again.", target)
		}
		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Frontline.Proxy.UpstreamHostUnreachable.URN(),
				fmt.Sprintf("The %s is unreachable. Please try again later or contact support at support@unkey.com.", target)
		}
	}

	// os.IsTimeout catches timeouts not surfaced as net.OpError (e.g.
	// http.Client deadlines reported as os.ErrDeadlineExceeded).
	if os.IsTimeout(err) {
		if isDialOp(err) {
			return codes.Frontline.Proxy.DialTimeout.URN(),
				fmt.Sprintf("The %s did not accept the connection in time. Please try again later.", target)
		}
		return codes.Frontline.Proxy.UpstreamResponseTimeout.URN(),
			fmt.Sprintf("The %s did not respond in time. Please try again later.", target)
	}

	// Fallback: unrecognised error. Default to proxy_error_unclassified
	// (frontline_fault outcome) so we get paged and learn what it is.
	return codes.Frontline.Proxy.ProxyErrorUnclassified.URN(),
		fmt.Sprintf("Failed to connect to the %s. Please try again or contact support at support@unkey.com.", target)
}

// classifyPeerFrontlineError maps a wire-level error talking to a peer
// frontline to its peer_frontline_* URN. Every URN produced here maps to
// OutcomeFrontlineFault in the metrics layer: we control peer frontlines,
// so any failure on that path is provably ours.
func classifyPeerFrontlineError(err error) (codes.URN, string) {
	const target = "peer frontline"

	// DNS resolution of peer-frontline hostnames runs on this path; check
	// before generic timeout/OpError handling because *net.DNSError also
	// satisfies os.IsTimeout (via Timeout()) and would otherwise land in
	// the generic peer_frontline_timeout bucket.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsTimeout {
			return codes.Frontline.Proxy.PeerFrontlineDNSTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
		return codes.Frontline.Proxy.PeerFrontlineDNSNotFound.URN(),
			fmt.Sprintf("DNS resolution failed for the %s. Please try again later or contact support at support@unkey.com.", target)
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return codes.Frontline.Proxy.PeerFrontlineTimeout.URN(),
				fmt.Sprintf("The %s did not respond in time. Please try again later.", target)
		}
		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Frontline.Proxy.PeerFrontlineConnectionRefused.URN(),
				fmt.Sprintf("The %s refused the connection. Please try again in a few seconds.", target)
		}
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Frontline.Proxy.PeerFrontlineConnectionReset.URN(),
				fmt.Sprintf("The %s reset the connection. Please try again.", target)
		}
		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(),
				fmt.Sprintf("The %s is unreachable. Please try again later or contact support at support@unkey.com.", target)
		}
	}

	if os.IsTimeout(err) {
		return codes.Frontline.Proxy.PeerFrontlineTimeout.URN(),
			fmt.Sprintf("The %s did not respond in time. Please try again later.", target)
	}

	// Fallback: unrecognised peer-frontline error. ProxyErrorUnclassified
	// is the right bucket — it's frontline_fault, and we'd rather learn
	// what it is than silently call it a peer-frontline issue.
	return codes.Frontline.Proxy.ProxyErrorUnclassified.URN(),
		fmt.Sprintf("Failed to connect to the %s. Please try again or contact support at support@unkey.com.", target)
}
