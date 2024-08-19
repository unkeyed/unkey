package ratelimit

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type (
	leaseId string
)
type lease struct {
	Cost      int64
	ExpiresAt time.Time
}

type ratelimitRequest struct {
	// Optionally set a time, to replay this request at a specific time on the origin node
	// defaults to time.Now() if not set
	Time       time.Time
	Name       string
	Identifier string
	Limit      int64
	Cost       int64
	Duration   time.Duration
	Lease      *lease
}

type windowResponse struct {
	// whether the window was created as part of the request
	created   bool
	bucketKey bucketKey
	sequence  int64
	counter   int64
}

type ratelimitResponse struct {
	Pass      bool
	Limit     int64
	Remaining int64
	Reset     int64
	Current   int64

	currentWindow  windowResponse
	previousWindow windowResponse
}

type setCounterRequest struct {
	Identifier string
	Limit      int64
	Duration   time.Duration
	Sequence   int64
	Counter    int64
	Time       time.Time
}

type commitLeaseRequest struct {
	Identifier string
	LeaseId    string
	Tokens     int64
}

type window struct {

	// when this window starts
	start time.Time

	// the duration of this window
	duration time.Duration

	counter int64

	// leaseId -> lease
	leases map[string]*lease
}

func (w *window) reset() time.Time {
	return w.start.Add(w.duration)

}

// Generally there is one bucket per identifier.
// However if the same identifier is used with different config, such as limit
// or duration, there will be multiple buckets for the same identifier.
//
// A bucket is always uniquely identified by this triplet: identifier, limit, duration.
// See `bucketKey` for more details.
//
// A bucket reaches its lifetime when the last window has expired at least 1 * duration ago.
// In other words, we can remove a bucket when it is no longer relevant for
// ratelimit decisions.
type bucket struct {
	sync.RWMutex
	limit    int64
	duration time.Duration
	// sequence -> window
	windows map[int64]*window
}

type slidingWindow struct {
	bucketsLock sync.RWMutex
	// identifier+sequence -> bucket
	buckets             map[string]*bucket
	leaseIdToKeyMapLock sync.RWMutex
	// Store a reference leaseId -> window key
	leaseIdToKeyMap map[string]string
	logger          logging.Logger
}

func NewSlidingWindow(logger logging.Logger) *slidingWindow {

	r := &slidingWindow{
		bucketsLock:         sync.RWMutex{},
		buckets:             make(map[string]*bucket),
		leaseIdToKeyMapLock: sync.RWMutex{},
		leaseIdToKeyMap:     make(map[string]string),
		logger:              logger,
	}

	repeat.Every(time.Minute, r.removeExpiredIdentifiers)
	return r

}

// bucketKey returns a unique key for an identifier and duration config
// the duration is required to ensure a change in ratelimit config will not
// reuse the same bucket and mess up the sequence numbers
type bucketKey struct {
	identifier string
	limit      int64
	duration   time.Duration
}

func (b bucketKey) ToString() string {
	return fmt.Sprintf("%s-%d-%d", b.identifier, b.limit, b.duration.Milliseconds())
}

// removeExpiredIdentifiers removes buckets that are no longer relevant
// for ratelimit decisions
func (r *slidingWindow) removeExpiredIdentifiers() {
	r.bucketsLock.Lock()
	defer r.bucketsLock.Unlock()

	activeRatelimits.Set(float64(len(r.buckets)))
	now := time.Now()
	for id, bucket := range r.buckets {
		bucket.Lock()
		for seq, w := range bucket.windows {
			if now.After(w.start.Add(2 * bucket.duration)) {
				delete(bucket.windows, seq)
			}
		}
		if len(bucket.windows) == 0 {
			delete(r.buckets, id)
		}
		bucket.Unlock()
	}
}

// CheckWindows returns whether the previous and current windows exist for the given request
func (r *slidingWindow) CheckWindows(ctx context.Context, req ratelimitRequest) (prev bool, curr bool) {
	ctx, span := tracing.Start(ctx, "slidingWindow.CheckWindows")
	defer span.End()

	if req.Time.IsZero() {
		req.Time = time.Now()
	}

	key := bucketKey{req.Identifier, req.Limit, req.Duration}

	r.bucketsLock.RLock()
	bucket, ok := r.buckets[key.ToString()]
	r.bucketsLock.RUnlock()
	if !ok {
		return false, false
	}

	currentWindowSequence := req.Time.UnixMilli() / req.Duration.Milliseconds()
	previousWindowSequence := currentWindowSequence - 1

	bucket.RLock()
	_, curr = bucket.windows[currentWindowSequence]
	_, prev = bucket.windows[previousWindowSequence]
	bucket.RUnlock()
	return prev, curr
}

func (r *slidingWindow) Take(ctx context.Context, req ratelimitRequest) ratelimitResponse {
	ctx, span := tracing.Start(ctx, "slidingWindow.Take")
	defer span.End()

	if req.Time.IsZero() {
		req.Time = time.Now()
	}

	key := bucketKey{req.Identifier, req.Limit, req.Duration}
	span.SetAttributes(attribute.String("key", string(key.ToString())))

	r.bucketsLock.RLock()
	bucket, ok := r.buckets[key.ToString()]
	r.bucketsLock.RUnlock()
	span.SetAttributes(attribute.Bool("identifierExisted", ok))
	if !ok {
		bucket = newBucket(req.Limit, req.Duration)
		r.bucketsLock.Lock()
		r.buckets[key.ToString()] = bucket
		r.bucketsLock.Unlock()
	}

	bucket.Lock()
	defer bucket.Unlock()

	currentWindowStart := req.Time.Truncate(req.Duration)
	previousWindowStart := currentWindowStart.Add(-req.Duration)

	currentWindowSequence := req.Time.UnixMilli() / req.Duration.Milliseconds()
	previousWindowSequence := currentWindowSequence - 1

	currentWindow, currentWindowExists := bucket.windows[currentWindowSequence]
	if !currentWindowExists {
		currentWindow = newWindow(currentWindowStart, req.Duration)
		bucket.windows[currentWindowSequence] = currentWindow
	}

	currentWindowPercentage := float64(req.Time.Sub(currentWindow.start).Milliseconds()) / float64(req.Duration.Milliseconds())
	previousWindowPercentage := 1.0 - currentWindowPercentage

	// Calculate the current count including all leases
	previousWindow, previousWindowExists := bucket.windows[previousWindowSequence]
	if !previousWindowExists {
		previousWindow = newWindow(previousWindowStart, req.Duration)
		bucket.windows[previousWindowSequence] = previousWindow
	}
	fromPreviousWindow := float64(previousWindow.counter) * previousWindowPercentage
	fromCurrentWindow := float64(currentWindow.counter)

	current := int64(math.Round(fromCurrentWindow + fromPreviousWindow))

	// r.logger.Info().Int64("fromCurrentWindow", fromCurrentWindow).Int64("fromPreviousWindow", fromPreviousWindow).Time("now", req.Time).Time("currentWindow.start", currentWindow.start).Int64("msSinceStart", msSinceStart).Float64("currentWindowPercentage", currentWindowPercentage).Float64("previousWindowPercentage", previousWindowPercentage).Bool("currentWindowExists", currentWindowExists).Bool("previousWindowExists", previousWindowExists).Int64("current", current).Interface("buckets", r.buckets).Send()
	// currentWithLeases := id.current
	// if req.Lease != nil {
	// 	currentWithLeases += req.Lease.Cost
	// }
	// for _, lease := range id.leases {
	// 	if lease.expiresAt.Before(time.Now()) {
	// 		currentWithLeases += lease.cost
	// 	}
	// }

	// Evaluate if the request should pass or not

	if current+req.Cost > req.Limit {
		ratelimitsCount.WithLabelValues("false").Inc()
		remaining := req.Limit - current
		if remaining < 0 {
			remaining = 0
		}
		return ratelimitResponse{
			Pass:      false,
			Remaining: remaining,
			Reset:     currentWindow.reset().UnixMilli(),
			Limit:     req.Limit,
			Current:   current,
			currentWindow: windowResponse{
				created:   !currentWindowExists,
				bucketKey: key,
				sequence:  currentWindowSequence,
				counter:   currentWindow.counter,
			},
			previousWindow: windowResponse{
				created:   !previousWindowExists,
				bucketKey: key,
				sequence:  previousWindowSequence,
				counter:   previousWindow.counter,
			},
		}
	}

	// if req.Lease != nil {
	// 	leaseId := uid.New("lease")
	// 	id.leases[leaseId] = lease{
	// 		id:        leaseId,
	// 		cost:      req.Lease.Cost,
	// 		expiresAt: req.Lease.ExpiresAt,
	// 	}
	// 	r.leaseIdToKeyMapLock.Lock()
	// 	r.leaseIdToKeyMap[leaseId] = key
	// 	r.leaseIdToKeyMapLock.Unlock()
	// }
	currentWindow.counter += req.Cost
	current += req.Cost

	remaining := req.Limit - current
	if remaining < 0 {
		remaining = 0
	}

	// currentWithLeases += req.Cost
	ratelimitsCount.WithLabelValues("true").Inc()
	return ratelimitResponse{
		Pass:      true,
		Remaining: remaining,
		Reset:     currentWindow.reset().UnixMilli(),
		Limit:     req.Limit,
		Current:   current,
		currentWindow: windowResponse{
			created:   !currentWindowExists,
			bucketKey: key,
			sequence:  currentWindowSequence,
			counter:   currentWindow.counter,
		},
		previousWindow: windowResponse{
			created:   !previousWindowExists,
			bucketKey: key,
			sequence:  previousWindowSequence,
			counter:   previousWindow.counter,
		},
	}
}

func (r *slidingWindow) SetCounter(ctx context.Context, requests ...setCounterRequest) error {
	ctx, span := tracing.Start(ctx, "slidingWindow.SetCounter")
	defer span.End()
	for _, req := range requests {
		key := bucketKey{req.Identifier, req.Limit, req.Duration}.ToString()
		r.bucketsLock.RLock()
		bucket, ok := r.buckets[key]
		r.bucketsLock.RUnlock()
		span.SetAttributes(attribute.Bool("identifierExisted", ok))

		if !ok {
			bucket = newBucket(req.Limit, req.Duration)
			r.bucketsLock.Lock()
			r.buckets[key] = bucket
			r.bucketsLock.Unlock()
		}

		// Only increment the current value if the new value is greater than the current value
		// Due to varying network latency, we may receive out of order responses and could decrement the
		// current value, which would result in inaccurate rate limiting
		bucket.Lock()
		window, ok := bucket.windows[req.Sequence]
		if !ok {
			window = newWindow(req.Time, req.Duration)
			bucket.windows[req.Sequence] = window
		}
		if req.Counter > window.counter {
			window.counter = req.Counter
		}
		bucket.Unlock()

	}
	return nil
}

func (r *slidingWindow) CommitLease(ctx context.Context, req commitLeaseRequest) error {
	// ctx, span := tracing.Start(ctx, "slidingWindow.SetCounter")
	// defer span.End()

	// r.leaseIdToKeyMapLock.RLock()
	// key, ok := r.leaseIdToKeyMap[req.LeaseId]
	// r.leaseIdToKeyMapLock.RUnlock()
	// if !ok {
	// 	r.logger.Warn().Str("leaseId", req.LeaseId).Msg("leaseId not found")
	// 	return nil
	// }

	// r.bucketsLock.Lock()
	// defer r.bucketsLock.Unlock()
	// window, ok := r.buckets[key]
	// if !ok {
	// 	r.logger.Warn().Str("key", key).Msg("key not found")
	// 	return nil
	// }

	// _, ok = window.leases[req.LeaseId]
	// if !ok {
	// 	r.logger.Warn().Str("leaseId", req.LeaseId).Msg("leaseId not found")
	// 	return nil
	// }

	return fmt.Errorf("not implemented")

}

func newBucket(limit int64, duration time.Duration) *bucket {
	return &bucket{
		limit:    limit,
		duration: duration,
		windows:  make(map[int64]*window),
	}
}

func newWindow(t time.Time, duration time.Duration) *window {
	return &window{
		start:    t.Truncate(duration),
		duration: duration,
		counter:  0,
		leases:   make(map[string]*lease),
	}
}
