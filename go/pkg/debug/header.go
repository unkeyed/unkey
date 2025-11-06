package debug

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CacheHeader represents a structured cache debug header
type CacheHeader struct {
	CacheName string
	Latency   time.Duration
	Status    string
}

// NewCacheHeader creates a new cache header with the given parameters
func NewCacheHeader(cacheName string, status string, latency time.Duration) CacheHeader {
	return CacheHeader{
		CacheName: cacheName,
		Latency:   latency,
		Status:    status,
	}
}

// String formats the cache header as a string in the format "cache_name:latency:status"
func (h CacheHeader) String() string {
	return fmt.Sprintf("%s:%s:%s", h.CacheName, formatDuration(h.Latency), h.Status)
}

// ParseCacheHeader parses a cache header string in the format "cache_name:latency:status"
func ParseCacheHeader(headerValue string) (CacheHeader, error) {
	parts := strings.Split(headerValue, ":")
	if len(parts) != 3 {
		return CacheHeader{}, fmt.Errorf("invalid cache header format: expected 3 parts separated by ':', got %d parts", len(parts))
	}

	cacheName := parts[0]
	latencyStr := parts[1]
	status := parts[2]

	// Parse latency
	latency, err := parseDuration(latencyStr)
	if err != nil {
		return CacheHeader{}, fmt.Errorf("failed to parse latency '%s': %w", latencyStr, err)
	}

	return CacheHeader{
		CacheName: cacheName,
		Latency:   latency,
		Status:    status,
	}, nil
}

// parseDuration parses duration strings in the format produced by formatDuration
func parseDuration(durationStr string) (time.Duration, error) {
	if strings.HasSuffix(durationStr, "ms") {
		msStr := strings.TrimSuffix(durationStr, "ms")
		ms, err := strconv.ParseFloat(msStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid millisecond value: %w", err)
		}
		return time.Duration(ms * float64(time.Millisecond)), nil
	}

	if strings.HasSuffix(durationStr, "us") {
		usStr := strings.TrimSuffix(durationStr, "us")
		us, err := strconv.ParseFloat(usStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid microsecond value: %w", err)
		}
		return time.Duration(us * float64(time.Microsecond)), nil
	}

	// Also support the legacy μs format for backward compatibility
	if strings.HasSuffix(durationStr, "μs") {
		usStr := strings.TrimSuffix(durationStr, "μs")
		us, err := strconv.ParseFloat(usStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid microsecond value: %w", err)
		}
		return time.Duration(us * float64(time.Microsecond)), nil
	}

	return 0, fmt.Errorf("unsupported duration format: expected 'ms', 'us', or 'μs' suffix")
}
