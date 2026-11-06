package watcher

import (
	"sync"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
)

const (
	retryDelayMin = 5 * time.Second
	retryDelayMax = 5 * time.Minute
)

// retryQueue holds the latest stream event that failed for each workload, so
// one failing workload does not hold back the stream for the whole region.
// The queue is lost on restart; the full sync that runs at startup covers it.
type retryQueue struct {
	mu      sync.Mutex
	pending map[string]pendingEvent
}

type pendingEvent struct {
	event    *ctrlv1.DeploymentChangeEvent
	attempts int
	due      time.Time
}

func newRetryQueue() *retryQueue {
	return &retryQueue{mu: sync.Mutex{}, pending: map[string]pendingEvent{}}
}

// add replaces the pending event for key and backs off its next attempt.
func (q *retryQueue) add(key string, event *ctrlv1.DeploymentChangeEvent, now time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.backoff(key, event, now)
}

// queueNow replaces the pending event for key and makes it due at once. The
// stream uses it when a retry of the same workload still runs.
func (q *retryQueue) queueNow(key string, event *ctrlv1.DeploymentChangeEvent, now time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending[key] = pendingEvent{event: event, attempts: q.pending[key].attempts, due: now}
	metrics.StreamRetriesPending.Set(float64(len(q.pending)))
}

func (q *retryQueue) remove(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.pending, key)
	metrics.StreamRetriesPending.Set(float64(len(q.pending)))
}

// fail backs off event again only if it is still the pending event for key,
// so a retry that ends cannot overwrite a newer event the stream queued.
func (q *retryQueue) fail(key string, event *ctrlv1.DeploymentChangeEvent, now time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.pending[key].event != event {
		return
	}
	q.backoff(key, event, now)
}

// finish removes event only if it is still the pending event for key, so a
// retry that ends cannot drop a newer event the stream queued.
func (q *retryQueue) finish(key string, event *ctrlv1.DeploymentChangeEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.pending[key].event != event {
		return
	}
	delete(q.pending, key)
	metrics.StreamRetriesPending.Set(float64(len(q.pending)))
}

func (q *retryQueue) backoff(key string, event *ctrlv1.DeploymentChangeEvent, now time.Time) {
	attempts := q.pending[key].attempts + 1
	q.pending[key] = pendingEvent{event: event, attempts: attempts, due: now.Add(retryDelay(attempts))}
	metrics.StreamRetriesPending.Set(float64(len(q.pending)))
}

// holds reports whether event is still the pending event for key. A newer
// stream event for the same workload replaces or removes it.
func (q *retryQueue) holds(key string, event *ctrlv1.DeploymentChangeEvent) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.pending[key].event == event
}

func (q *retryQueue) due(now time.Time) map[string]*ctrlv1.DeploymentChangeEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	due := map[string]*ctrlv1.DeploymentChangeEvent{}
	for key, pending := range q.pending {
		if !now.Before(pending.due) {
			due[key] = pending.event
		}
	}
	return due
}

func retryDelay(attempts int) time.Duration {
	delay := retryDelayMin
	for range attempts - 1 {
		delay *= 2
		if delay >= retryDelayMax {
			return retryDelayMax
		}
	}
	return delay
}
