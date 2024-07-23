package metrics

import "time"

type HttpRequest struct {
	Method         string `json:"method"`
	Path           string `json:"path"`
	ServiceLatency int64  `json:"serviceLatency"`
	UserAgent      string `json:"userAgent"`
	RemoteAddr     string `json:"remoteAddr"`
	SourceIP       string `json:"sourceIP"`
}

func (m HttpRequest) Name() string {
	return "metric.http.request"
}

type CacheHit struct {
	Key      string `json:"key"`
	Hit      bool   `json:"hit"`
	Resource string `json:"resource"`
	Latency  int64  `json:"latency" `
	Tier     string `json:"tier"`
}

func (m CacheHit) Name() string {
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

func (m CacheHealth) Name() string {
	return "metric.cache.health"
}

type CacheEviction struct {
	Stale time.Time `json:"stale"`
	Now   time.Time `json:"now"`
	Key   string    `json:"key"`
}

func (m CacheEviction) Name() string {
	return "metric.cache.eviction"
}

type RingState struct {
	Nodes  int    `json:"nodes"`
	Tokens int    `json:"tokens"`
	State  string `json:"state"`
}

func (m RingState) Name() string {
	return "metric.ring.state"
}

type EventRouterFlushes struct {
	Rows int `json:"rows"`
}

func (m EventRouterFlushes) Name() string {
	return "metric.eventrouter.flushes"
}

type SystemLoad struct {
	CpuUsage float64 `json:"cpuUsage"`
	Memory   struct {
		Percentage float64 `json:"percentage"`
		Used       uint64  `json:"used"`
		Total      uint64  `json:"total"`
	} `json:"memory"`
}

func (m SystemLoad) Name() string {
	return "metric.system.load"
}

type ClusterSize struct {
	Size int `json:"size"`
}

func (m ClusterSize) Name() string {
	return "metric.cluster.size"
}

type ChannelBuffer struct {
	ID      string `json:"id"`
	Size    int    `json:"size"`
	MaxSize int    `json:"maxSize"`
}

func (m ChannelBuffer) Name() string {
	return "metric.channel.buffer"
}
