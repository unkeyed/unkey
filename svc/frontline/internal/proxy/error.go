package proxy

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"

	"fmt"

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

// categorizeProxyError maps a raw upstream / dial error into a stable
// codes.URN plus a public-facing message. The URN drives status code
// selection in middleware; the message is what the client sees in the
// rendered error page or JSON body. target is the human-readable name of
// what we failed to reach (e.g. "deployment instance", "peer frontline")
// and is interpolated into the message.
//
// Callers should preserve any pre-existing fault code via fault.GetCode
// before calling this — it has no way to know whether err was already
// classified upstream.
func categorizeProxyError(err error, target string) (codes.URN, string) {
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Frontline.Proxy.GatewayTimeout.URN(),
			fmt.Sprintf("The %s did not respond in time. Please try again later.", target)
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return codes.Frontline.Proxy.GatewayTimeout.URN(),
				fmt.Sprintf("The %s did not respond in time. Please try again later.", target)
		}

		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Frontline.Proxy.ServiceUnavailable.URN(),
				fmt.Sprintf("The %s refused the connection. It may be restarting — please try again in a few seconds.", target)
		}

		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Frontline.Proxy.BadGateway.URN(),
				fmt.Sprintf("The %s reset the connection unexpectedly. Please try again.", target)
		}

		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Frontline.Proxy.ServiceUnavailable.URN(),
				fmt.Sprintf("The %s is unreachable. Please try again later or contact support at support@unkey.com.", target)
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return codes.Frontline.Proxy.ServiceUnavailable.URN(),
				fmt.Sprintf("DNS resolution failed for the %s. Please check your configuration or contact support at support@unkey.com.", target)
		}
		if dnsErr.IsTimeout {
			return codes.Frontline.Proxy.GatewayTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
	}

	return codes.Frontline.Proxy.BadGateway.URN(),
		fmt.Sprintf("Failed to connect to the %s. Please try again or contact support at support@unkey.com.", target)
}
