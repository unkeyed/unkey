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
