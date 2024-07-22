package metrics

type Metrics interface {
	Record(metric Metric)
	Close()
}

// Metric is the interface that all metrics must implement to be recorded by the metrics package
//
// A metric must have a name that is unique within the system
// The remaining public fields are up to the caller and will be serialized to JSON when recorded
type Metric interface {
	// The name of the metric
	// e.g. "metric.cache.hit"
	Name() string
}

type metricWithBase struct {
	Metric
	Base
}

type Base struct {
	Metric      string
	Time        int64
	AxiomTime   int64 `json:"_time"` // _time is a special field in axiom
	NodeId      string
	Region      string
	Application string
}
