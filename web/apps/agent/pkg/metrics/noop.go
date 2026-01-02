package metrics

type noop struct {
}

func NewNoop() Metrics {
	return &noop{}

}

func (n *noop) Close() {}

func (n *noop) Record(metric Metric) {}
