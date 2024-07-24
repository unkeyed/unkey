package cache

type CacheStats struct {
	// How many set requests were rejected because the cache is full
	Rejected int64 `json:"rejected"`

	// Items in the cache
	Size int `json:"size"`

	// What is being stored in the cache
	Resource string `json:"resource"`
}

func (CacheStats) Name() string {
	return "metric.cache.stats"
}
