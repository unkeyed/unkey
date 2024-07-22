package metrics

import "time"

type HttpRequest struct {
	Method         string
	Path           string
	ServiceLatency int64
	UserAgent      string
	RemoteAddr     string
	SourceIP       string
}

func (m HttpRequest) Name() string {
	return "metric.http.request"
}

type CacheHit struct {
	Key      string
	Hit      bool
	Resource string
	Latency  int64
	Tier     string
}

func (m CacheHit) Name() string {
	return "metric.cache.hit"
}

type CacheHealth struct {
	CacheSize        int
	CacheMaxSize     int
	LruSize          int
	RefreshQueueSize int
	Utilization      float64
	Resource         string
	Tier             string
}

func (m CacheHealth) Name() string {
	return "metric.cache.health"
}

type CacheEviction struct {
	Stale time.Time
	Now   time.Time
	Key   string
}

func (m CacheEviction) Name() string {
	return "metric.cache.eviction"
}

type RingState struct {
	Nodes  int
	Tokens int
	State  string
}

func (m RingState) Name() string {
	return "metric.ring.state"
}

type EventRouterFlushes struct {
	Rows int
}

func (m EventRouterFlushes) Name() string {
	return "metric.eventrouter.flushes"
}

type SystemLoad struct {
	CpuUsage float64
	Memory   struct {
		Percentage float64
		Used       uint64
		Total      uint64
	}
}

func (m SystemLoad) Name() string {
	return "metric.system.load"
}

type ClusterSize struct {
	Size int
}

func (m ClusterSize) Name() string {
	return "metric.cluster.size"
}

type ChannelBuffer struct {
	ID      string
	Size    int
	MaxSize int
}

func (m ChannelBuffer) Name() string {
	return "metric.channel.buffer"
}
