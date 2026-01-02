package proxy

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"

	"github.com/unkeyed/unkey/go/pkg/codes"
)

func categorizeProxyError(err error) (codes.URN, string) {
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Frontline.Proxy.GatewayTimeout.URN(),
			"The request took too long to process. Please try again later."
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return codes.Frontline.Proxy.GatewayTimeout.URN(),
				"The request took too long to process. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Frontline.Proxy.ServiceUnavailable.URN(),
				"The service is temporarily unavailable. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Frontline.Proxy.BadGateway.URN(),
				"Connection was reset by the backend service. Please try again."
		}

		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Frontline.Proxy.ServiceUnavailable.URN(),
				"The service is unreachable. Please try again later."
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return codes.Frontline.Proxy.ServiceUnavailable.URN(),
				"The service could not be found. Please check your configuration."
		}
		if dnsErr.IsTimeout {
			return codes.Frontline.Proxy.GatewayTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
	}

	return codes.Frontline.Proxy.BadGateway.URN(),
		"Unable to connect to. Please try again in a few moments."
}
