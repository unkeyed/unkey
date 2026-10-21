package proxy

import (
	"context"
	"errors"
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
	if netErr, ok := errors.AsType[*net.OpError](err); ok && netErr.Op == "dial" {
		return true
	}
	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return true
	}
	return false
}

func categorizeProxyError(err error, destination proxyDestination) (codes.URN, string) {
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	if destination == destinationFrontline {
		return codes.Frontline.Internal.InternalServerError.URN(),
			"Unkey could not reach the serving region. Try again later."
	}

	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Frontline.Proxy.GatewayTimeout.URN(),
			"The application did not respond in time. Please try again later."
	}

	if netErr, ok := errors.AsType[*net.OpError](err); ok {
		if netErr.Timeout() {
			return codes.Frontline.Proxy.GatewayTimeout.URN(),
				"The application did not respond in time. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Frontline.Proxy.BadGateway.URN(),
				"The application refused the connection. Check that it is running and listening on the configured port on 0.0.0.0, not only localhost."
		}

		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Frontline.Proxy.BadGateway.URN(),
				"The application reset the connection unexpectedly. Please try again."
		}

		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Frontline.Proxy.ServiceUnavailable.URN(),
				"The application is unreachable. Please try again later or contact support at support@unkey.com."
		}
	}

	if dnsErr, ok := errors.AsType[*net.DNSError](err); ok {
		if dnsErr.IsNotFound {
			return codes.Frontline.Proxy.ServiceUnavailable.URN(),
				"DNS resolution failed for the application. Please check your configuration or contact support at support@unkey.com."
		}
		if dnsErr.IsTimeout {
			return codes.Frontline.Proxy.GatewayTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
	}

	return codes.Frontline.Proxy.BadGateway.URN(),
		"Failed to connect to the application. Please try again or contact support at support@unkey.com."
}
