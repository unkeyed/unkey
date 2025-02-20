package otel

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

var globalMetrics metric.MeterProvider

func init() {
	globalMetrics = noop.NewMeterProvider()
}

func GetGlobalMeterProvider() metric.MeterProvider {
	return globalMetrics
}



var (
	MetricHttpRequests = globalMetrics.Meter(name string, opts ...metric.MeterOption)
)
