package proxy

import "github.com/unkeyed/unkey/go/pkg/otel/logging"

// SingleTargetProxy creates a proxy that always forwards to a single target.
func SingleTargetProxy(targetURL string, logger logging.Logger) (Proxy, error) {
	return New(Config{
		Targets: []string{targetURL},
		Logger:  logger,
	})
}
