package metrics

type Metrics interface {
	ReportCacheHealth(CacheHealthReport)
	ReportDatabaseLatency(DatabaseLatencyReport)
	ReportCacheHit(CacheHitReport)
	Close()
}

type metricId string

const (
	httpRequest     metricId = "metric.http.request"
	keyVerifying    metricId = "metric.key.verification"
	cacheHealth     metricId = "metric.cache.health"
	databaseLatency metricId = "metric.database.latency"
	cacheHit        metricId = "metric.cache.hit"
)

type CacheHitReport struct {
	Key      string `json:"key"`
	Hit      bool   `json:"hit"`
	Resource string `json:"resource"`
	Latency  int64  `json:"latency"`
	Tier     string `json:"tier"`
}

type CacheHealthReport struct {
	CacheSize        int     `json:"cacheSize"`
	CacheMaxSize     int     `json:"cacheMaxSize"`
	LruSize          int     `json:"lruSize"`
	RefreshQueueSize int     `json:"refreshQueueSize"`
	Utilization      float64 `json:"utilization"`
	Resource         string  `json:"resource"`
	Tier             string  `json:"tier"`
}

type DatabaseLatencyReport struct {
	Query   string `json:"query"`
	Latency int64  `json:"latency"`
}
