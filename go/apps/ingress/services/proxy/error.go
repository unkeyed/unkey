package proxy

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"

	"github.com/unkeyed/unkey/go/pkg/codes"
)

// categorizeProxyError determines the appropriate error code and message based on the error type
func categorizeProxyError(err error) (codes.URN, string) {
	// Check for client-side cancellation (client closed connection)
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Ingress.Proxy.GatewayTimeout.URN(),
			"The request took too long to process. Please try again later."
	}

	// Check for network errors
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		// Check for timeout
		if netErr.Timeout() {
			return codes.Ingress.Proxy.GatewayTimeout.URN(),
				"The request took too long to process. Please try again later."
		}

		// Check for connection refused
		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Ingress.Proxy.ServiceUnavailable.URN(),
				"The service is temporarily unavailable. Please try again later."
		}

		// Check for connection reset
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Ingress.Proxy.BadGateway.URN(),
				"Connection was reset by the backend service. Please try again."
		}

		// Check for no route to host
		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Ingress.Proxy.ServiceUnavailable.URN(),
				"The service is unreachable. Please try again later."
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return codes.Ingress.Proxy.ServiceUnavailable.URN(),
				"The service could not be found. Please check your configuration."
		}

		if dnsErr.IsTimeout {
			return codes.Ingress.Proxy.GatewayTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
	}

	// Default to bad gateway
	return codes.Ingress.Proxy.BadGateway.URN(),
		"Unable to connect to the backend service. Please try again in a few moments."
}
