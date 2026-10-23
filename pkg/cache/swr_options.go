package cache

import "time"

// SWROption lowers how old a value one [Cache.SWR] read accepts. It never
// raises the limits the cache was configured with.
type SWROption func(*swrOptions)

type swrOptions struct {
	maxAgeFresh time.Duration
	maxAgeStale time.Duration
}

// MaxAgeFresh returns a cached value without revalidating it only while it is
// younger than d. Without [MaxAgeStale], it also bounds how old a value the read
// serves at all, so the read waits for the origin once the value is d old.
// A d of zero or less has no effect.
func MaxAgeFresh(d time.Duration) SWROption {
	return func(o *swrOptions) { o.maxAgeFresh = d }
}

// MaxAgeStale serves a cached value, revalidating it in the background, only
// while it is younger than d. Once it is older, the read waits for the origin.
// A d of zero or less has no effect.
func MaxAgeStale(d time.Duration) SWROption {
	return func(o *swrOptions) { o.maxAgeStale = d }
}

func newSWROptions(opts []SWROption) swrOptions {
	var o swrOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxAgeStale <= 0 {
		o.maxAgeStale = o.maxAgeFresh
	}
	return o
}

// limits returns when e stops being fresh and stops being servable for a read
// with o. A stale limit below the fresh limit lowers the fresh limit, because
// raising the stale limit would serve values older than the caller allowed.
func limits[V any](e swrEntry[V], o swrOptions) (fresh, stale time.Time) {
	fresh, stale = e.Fresh, e.Stale
	if o.maxAgeFresh > 0 {
		fresh = minTime(fresh, e.LoadedAt.Add(o.maxAgeFresh))
	}
	if o.maxAgeStale > 0 {
		stale = minTime(stale, e.LoadedAt.Add(o.maxAgeStale))
	}
	return minTime(fresh, stale), stale
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}
