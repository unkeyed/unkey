package middleware

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	handler "github.com/unkeyed/unkey/svc/sentinel/routes/proxy"
)

// WithProxyErrorHandling categorizes proxy errors and sets response status for logging.
func WithProxyErrorHandling() zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			err := next(ctx, s)
			if err == nil {
				return nil
			}

			tracking, ok := handler.SentinelTrackingFromContext(ctx)
			if !ok {
				return err
			}

			// Set response status for CH logging if not set by ModifyResponse
			if tracking.ResponseStatus == 0 {
				if errors.Is(err, context.Canceled) {
					tracking.ResponseStatus = 499 // Client Closed Request like nginx
				} else if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
					tracking.ResponseStatus = 504 // Gateway Timeout
				} else {
					tracking.ResponseStatus = 502 // Bad Gateway
				}
			}

			// Categorize error and wrap with instance context
			urn, message := categorizeProxyError(err)

			var instanceAddress string
			if tracking.Instance != nil {
				instanceAddress = tracking.Instance.Address
			}

			return fault.Wrap(err,
				fault.Code(urn),
				fault.Internal(fmt.Sprintf("proxy error forwarding to instance %s", instanceAddress)),
				fault.Public(message),
			)
		}
	}
}

func categorizeProxyError(err error) (codes.URN, string) {
	if errors.Is(err, context.Canceled) {
		return codes.User.BadRequest.ClientClosedRequest.URN(),
			"The client closed the connection before the request completed."
	}

	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return codes.Sentinel.Proxy.SentinelTimeout.URN(),
			"The request took too long to process. Please try again later."
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return codes.Sentinel.Proxy.SentinelTimeout.URN(),
				"The request took too long to process. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return codes.Sentinel.Proxy.ServiceUnavailable.URN(),
				"The service is temporarily unavailable. Please try again later."
		}

		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return codes.Sentinel.Proxy.BadGateway.URN(),
				"Connection was reset by the backend service. Please try again."
		}

		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return codes.Sentinel.Proxy.ServiceUnavailable.URN(),
				"The service is unreachable. Please try again later."
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return codes.Sentinel.Proxy.ServiceUnavailable.URN(),
				"The service could not be found. Please check your configuration."
		}
		if dnsErr.IsTimeout {
			return codes.Sentinel.Proxy.SentinelTimeout.URN(),
				"DNS resolution timed out. Please try again later."
		}
	}

	return codes.Sentinel.Proxy.BadGateway.URN(),
		"Unable to connect to an instance. Please try again in a few moments."
}
