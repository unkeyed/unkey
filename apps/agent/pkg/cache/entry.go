package cache

import (
	"container/list"
	"time"
)

type swrEntry[T any] struct {
	Value T `json:"value"`

	Hit CacheHit `json:"hit"`
	// Before this time the entry is considered fresh and vaid
	Fresh time.Time `json:"fresh"`
	// Before this time, the entry should be revalidated
	// After this time, the entry must be discarded
	Stale      time.Time     `json:"stale"`
	LruElement *list.Element `json:"-"`
}
