package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type Int64Counter interface {
	Add(ctx context.Context, incr int64, options ...metric.AddOption)
}

type Int64ObservableGauge interface {
	RegisterCallback(metric.Int64Callback) error
}
