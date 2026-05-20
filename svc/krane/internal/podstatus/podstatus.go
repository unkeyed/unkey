// Package podstatus provides small helpers for inspecting a pod's
// ContainersReady condition from krane's watch loops. Both the deployment
// and sentinel controllers use them to emit the same lag metric and log
// fields.
package podstatus

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// LagRecorder records time-since-ContainersReady-True as a watch delivery
// lag sample on [metrics.PodWatchDeliveryLagSeconds], deduplicated per
// (pod UID, transition time) so repeat events for the same transition
// don't re-record pod age.
//
// Without the dedup, any Modified event that arrives long after the pod
// first became ready (annotation churn, status updates on unrelated
// fields) would re-sample the static LastTransitionTime and skew the
// histogram toward "pod age", not delivery lag. Deleted events are
// skipped entirely for the same reason: the LastTransitionTime on a
// deleted pod is historical.
type LagRecorder struct {
	// component labels the metric ("deployment" or "sentinel").
	component string

	// observed caches (pod UID -> last recorded LastTransitionTime) so we
	// only emit one sample per actual transition. Nil disables dedup and
	// falls back to recording every eligible event.
	observed cache.Cache[string, time.Time]
}

// NewLagRecorder returns a recorder that writes samples under the given
// component label. observed is a bounded cache used for per-transition
// dedup; passing nil disables dedup (samples are still filtered to
// non-Deleted events with a real LastTransitionTime).
func NewLagRecorder(component string, observed cache.Cache[string, time.Time]) *LagRecorder {
	return &LagRecorder{component: component, observed: observed}
}

// Observe records a lag sample for the pod's ContainersReady=True
// condition, if one exists and we haven't already recorded this exact
// transition. Deleted events and pods without a condition/transition time
// are silently skipped.
func (r *LagRecorder) Observe(ctx context.Context, pod *corev1.Pod, eventType watch.EventType) {
	if r == nil || pod == nil || eventType == watch.Deleted {
		return
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type != corev1.ContainersReady || cond.Status != corev1.ConditionTrue {
			continue
		}
		if cond.LastTransitionTime.IsZero() {
			return
		}

		uid := string(pod.UID)
		if uid != "" && r.observed != nil {
			if prev, hit := r.observed.Get(ctx, uid); hit == cache.Hit && prev.Equal(cond.LastTransitionTime.Time) {
				return
			}
		}

		lag := time.Since(cond.LastTransitionTime.Time).Seconds()
		if lag < 0 {
			return
		}
		metrics.PodWatchDeliveryLagSeconds.WithLabelValues(r.component, "watch").Observe(lag)

		if uid != "" && r.observed != nil {
			r.observed.Set(ctx, uid, cond.LastTransitionTime.Time)
		}
		return
	}
}

// ReadyStatus returns the pod's ContainersReady condition as a short
// string ("true"/"false"/"unknown"/"absent") for log lines.
func ReadyStatus(pod *corev1.Pod) string {
	if pod == nil {
		return "absent"
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type != corev1.ContainersReady {
			continue
		}
		switch cond.Status {
		case corev1.ConditionTrue:
			return "true"
		case corev1.ConditionFalse:
			return "false"
		case corev1.ConditionUnknown:
			return "unknown"
		default:
			return "unknown"
		}
	}
	return "absent"
}

// ReadyLagSeconds returns the time since kubelet flipped
// ContainersReady=True on the pod, or -1 when unknown. Used as a log
// field so tailing krane shows how fresh each watch event is relative
// to the actual transition.
func ReadyLagSeconds(pod *corev1.Pod) float64 {
	if pod == nil {
		return -1
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type != corev1.ContainersReady || cond.Status != corev1.ConditionTrue {
			continue
		}
		if cond.LastTransitionTime.IsZero() {
			return -1
		}
		return time.Since(cond.LastTransitionTime.Time).Seconds()
	}
	return -1
}
