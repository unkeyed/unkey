package metrics

type Metrics interface {
	ReportHttpRequest(HttpRequestReport)
	ReportKeyVerification(KeyVerificationReport)
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

type HttpRequestReport struct {
	Path           string `json:"path"`
	Method         string `json:"method"`
	Status         int    `json:"status"`
	Error          string `json:"error"`
	ServiceLatency int64  `json:"serviceLatency"`
	EdgeRegion     string `json:"edgeRegion"`
	TraceId        string `json:"traceId"`
}

type KeyVerificationReport struct {
	KeyId       string `json:"keyId"`
	ApiId       string `json:"apiId"`
	KeyAuthId   string `json:"keyAuthId"`
	WorkspaceId string `json:"workspaceId"`

	TraceId string `json:"traceId"`
}

type CacheHitReport struct {
	Key      string `json:"key"`
	Hit      bool   `json:"hit"`
	Resource string `json:"resource"`
	Latency  int64  `json:"latency"`
}

type CacheHealthReport struct {
	CacheSize        int     `json:"cacheSize"`
	CacheMaxSize     int     `json:"cacheMaxSize"`
	LruSize          int     `json:"lruSize"`
	RefreshQueueSize int     `json:"refreshQueueSize"`
	Utilization      float64 `json:"utilization"`
}

type DatabaseLatencyReport struct {
	Query   string `json:"query"`
	Latency int64  `json:"latency"`
}
