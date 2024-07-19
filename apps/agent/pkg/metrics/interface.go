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
	Metric() string
}

type HttpRequestReport struct {
}

type CacheHit struct {
	Key      string `json:"key"`
	Hit      bool   `json:"hit"`
	Resource string `json:"resource"`
	Latency  int64  `json:"latency"`
	Tier     string `json:"tier"`
}

func (m CacheHit) Metric() string {
	return "metric.cache.hit"

}

type CacheHealth struct {
	CacheSize        int     `json:"cacheSize"`
	CacheMaxSize     int     `json:"cacheMaxSize"`
	LruSize          int     `json:"lruSize"`
	RefreshQueueSize int     `json:"refreshQueueSize"`
	Utilization      float64 `json:"utilization"`
	Resource         string  `json:"resource"`
	Tier             string  `json:"tier"`
}

func (m CacheHealth) Metric() string {
	return "metric.cache.health"
}

type SystemLoad struct {
	CpuUsage float64 `json:"cpuUsage"`
	Memory   struct {
		Percentage float64 `json:"percentage"`
		Used       uint64  `json:"used"`
		Total      uint64  `json:"total"`
	} `json:"memory"`
}

func (m SystemLoad) Metric() string {
	return "metric.system.load"
}
