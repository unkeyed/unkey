/*
Package prometheus provides utilities for exposing Prometheus metrics over HTTP.

This package makes it easy to integrate Prometheus metrics collection and exposure
into applications built with the zen framework. It handles the setup of a metrics
endpoint that Prometheus can scrape to collect runtime metrics from your application.

Common use cases include:
  - Creating a dedicated metrics server separate from your application server
  - Adding metrics endpoints to existing zen-based applications
  - Setting up consistent metrics across multiple microservices

This package is designed to work seamlessly with the zen framework while providing
all the functionality of the standard Prometheus client library.
*/
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/pkg/zen"
)

// NewWithRegistry creates a zen server that exposes metrics from a custom
// prometheus.Registry at the /metrics endpoint. Use this to control exactly
// which metrics each service exposes.
func NewWithRegistry(reg *prometheus.Registry) (*zen.Server, error) {
	z, err := zen.New(zen.Config{
		MaxRequestBodySize: 0,
		Flags:              nil,
		TLS:                nil,
		EnableH2C:          false,
		ReadTimeout:        0,
		WriteTimeout:       0,
	})
	if err != nil {
		return nil, err
	}

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	z.RegisterRoute([]zen.Middleware{}, zen.NewRoute("GET", "/metrics", func(ctx context.Context, s *zen.Session) error {
		h.ServeHTTP(s.ResponseWriter(), s.Request())
		return nil
	}))

	return z, nil
}
