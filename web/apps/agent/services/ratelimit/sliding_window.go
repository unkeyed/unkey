package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/svc/agent/pkg/tracing"
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

type ratelimitResponse struct {
	Pass      bool
	Limit     int64
	Remaining int64
	Reset     int64
	Current   int64

	currentWindow  *ratelimitv1.Window
	previousWindow *ratelimitv1.Window
}

type setCounterRequest struct {
	Identifier string
	Limit      int64
	Duration   time.Duration
	Sequence   int64
	// any time within the window
	Time    time.Time
	Counter int64
}

type commitLeaseRequest struct {
	Identifier string
	LeaseId    string
	Tokens     int64
}

// removeExpiredIdentifiers removes buckets that are no longer relevant
// for ratelimit decisions
func (r *service) removeExpiredIdentifiers() {
	r.bucketsMu.Lock()
	defer r.bucketsMu.Unlock()

	activeRatelimits.Set(float64(len(r.buckets)))
	now := time.Now()
	for id, bucket := range r.buckets {
		bucket.Lock()
		for seq, w := range bucket.windows {
			if now.UnixMilli() > (w.Start + 2*w.Duration) {
				delete(bucket.windows, seq)
			}
		}
		if len(bucket.windows) == 0 {
			delete(r.buckets, id)
		}
		bucket.Unlock()
	}
}

func calculateSequence(t time.Time, duration time.Duration) int64 {
	return t.UnixMilli() / duration.Milliseconds()
}

// CheckWindows returns whether the previous and current windows exist for the given request
func (r *service) CheckWindows(ctx context.Context, req ratelimitRequest) (prev bool, curr bool) {
	ctx, span := tracing.Start(ctx, "slidingWindow.CheckWindows")
	defer span.End()

	if req.Time.IsZero() {
		req.Time = time.Now()
	}

	key := bucketKey{req.Identifier, req.Limit, req.Duration}
	bucket, existedBefore := r.getBucket(key)
	if !existedBefore {
		return false, false
	}

	currentWindowSequence := calculateSequence(req.Time, req.Duration)
	previousWindowSequence := currentWindowSequence - 1

	bucket.RLock()
	_, curr = bucket.windows[currentWindowSequence]
	_, prev = bucket.windows[previousWindowSequence]
	bucket.RUnlock()
	return prev, curr
}

// ::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::
// ::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::
// Experimentally, we are reverting this to fixed-window until we can get rid
// of the cloudflare cachelayer.
//
// Throughout this function there is commented out and annotated code that we
// need to reenable later. Such code is also marked with the comment "FIXED-WINDOW"
// ::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::
// ::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::
func (r *service) Take(ctx context.Context, req ratelimitRequest) ratelimitResponse {
	ctx, span := tracing.Start(ctx, "slidingWindow.Take")
	defer span.End()

	if req.Time.IsZero() {
		req.Time = time.Now()
	}

	key := bucketKey{req.Identifier, req.Limit, req.Duration}
	span.SetAttributes(attribute.String("key", string(key.toString())))

	bucket, _ := r.getBucket(key)

	bucket.Lock()
	defer bucket.Unlock()

	currentWindow := bucket.getCurrentWindow(req.Time)
	previousWindow := bucket.getPreviousWindow(req.Time)
	// FIXED-WINDOW
	// uncomment
	// currentWindowPercentage := float64(req.Time.UnixMilli()-currentWindow.Start) / float64(req.Duration.Milliseconds())
	// previousWindowPercentage := 1.0 - currentWindowPercentage

	// Calculate the current count including all leases
	// FIXED-WINDOW
	// uncomment
	// fromPreviousWindow := float64(previousWindow.Counter) * previousWindowPercentage
	// fromCurrentWindow := float64(currentWindow.Counter)

	// FIXED-WINDOW
	// replace this with the following line
	// current := int64(math.Ceil(fromCurrentWindow + fromPreviousWindow))
	current := currentWindow.Counter

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
			Pass:           false,
			Remaining:      remaining,
			Reset:          currentWindow.Start + currentWindow.Duration,
			Limit:          req.Limit,
			Current:        current,
			currentWindow:  currentWindow,
			previousWindow: previousWindow,
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
	currentWindow.Counter += req.Cost
	if currentWindow.Counter >= req.Limit && !currentWindow.MitigateBroadcasted && r.mitigateBuffer != nil {
		currentWindow.MitigateBroadcasted = true
		r.mitigateBuffer <- mitigateWindowRequest{
			identifier: req.Identifier,
			limit:      req.Limit,
			duration:   req.Duration,
			window:     currentWindow,
		}
	}

	current += req.Cost

	remaining := req.Limit - current
	if remaining < 0 {
		remaining = 0
	}
	// currentWithLeases += req.Cost
	ratelimitsCount.WithLabelValues("true").Inc()
	return ratelimitResponse{
		Pass:           true,
		Remaining:      remaining,
		Reset:          currentWindow.Start + currentWindow.Duration,
		Limit:          req.Limit,
		Current:        current,
		currentWindow:  currentWindow,
		previousWindow: previousWindow,
	}
}

func (r *service) SetCounter(ctx context.Context, requests ...setCounterRequest) error {
	ctx, span := tracing.Start(ctx, "slidingWindow.SetCounter")
	defer span.End()
	for _, req := range requests {
		key := bucketKey{req.Identifier, req.Limit, req.Duration}
		bucket, _ := r.getBucket(key)

		// Only increment the current value if the new value is greater than the current value
		// Due to varying network latency, we may receive out of order responses and could decrement the
		// current value, which would result in inaccurate rate limiting
		bucket.Lock()
		window, ok := bucket.windows[req.Sequence]
		if !ok {
			window = newWindow(req.Sequence, req.Time, req.Duration)
			bucket.windows[req.Sequence] = window
		}
		if req.Counter > window.Counter {
			window.Counter = req.Counter
		}
		bucket.Unlock()

	}
	return nil
}

// func (r *service) commitLease(ctx context.Context, req commitLeaseRequest) error {
// 	ctx, span := tracing.Start(ctx, "slidingWindow.SetCounter")
// 	defer span.End()

// 	r.leaseIdToKeyMapLock.RLock()
// 	key, ok := r.leaseIdToKeyMap[req.LeaseId]
// 	r.leaseIdToKeyMapLock.RUnlock()
// 	if !ok {
// 		r.logger.Warn().Str("leaseId", req.LeaseId).Msg("leaseId not found")
// 		return nil
// 	}

// 	r.bucketsMu.Lock()
// 	defer r.bucketsMu.Unlock()
// 	window, ok := r.buckets[key]
// 	if !ok {
// 		r.logger.Warn().Str("key", key).Msg("key not found")
// 		return nil
// 	}

// 	_, ok = window.leases[req.LeaseId]
// 	if !ok {
// 		r.logger.Warn().Str("leaseId", req.LeaseId).Msg("leaseId not found")
// 		return nil
// 	}

// 	return fmt.Errorf("not implemented")

// }

func newWindow(sequence int64, t time.Time, duration time.Duration) *ratelimitv1.Window {
	return &ratelimitv1.Window{
		Sequence:            sequence,
		MitigateBroadcasted: false,
		Start:               t.Truncate(duration).UnixMilli(),
		Duration:            duration.Milliseconds(),
		Counter:             0,
		Leases:              make(map[string]*ratelimitv1.Lease),
	}
}
