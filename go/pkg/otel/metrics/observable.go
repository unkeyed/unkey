package metrics

import "go.opentelemetry.io/otel/metric"

type int64ObservableGauge struct {
	m    metric.Meter
	name string
	opts []metric.Int64ObservableGaugeOption
}

var _ Int64ObservableGauge = (*int64ObservableGauge)(nil)

func (g *int64ObservableGauge) RegisterCallback(callback metric.Int64Callback) error {
	_, err := g.m.Int64ObservableGauge(g.name, append(g.opts, metric.WithInt64Callback(callback))...)

	return err
}
