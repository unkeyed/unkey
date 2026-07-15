// Container lifecycle event capture: walks pod.Status, mirrors each
// container's corev1.ContainerState into an InstanceEvent.state oneof, and
// ships the events to ctrl via ReportInstanceEvents. Surfaces user-actionable
// failures (OOMKilled, exit codes, crashloops) the gateway can't currently
// report, and lets the logs viewer draw lifecycle dividers between runs.
//
// This runs alongside reportDeploymentStatus on every pod-watch tick.
// reportDeploymentStatus produces a coarse instance summary (Running /
// Pending / Failed); this file produces fine-grained per-container life
// events for the dashboard timeline.

package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
)

const (
	// eventKind* mirror the InstanceEvent.state oneof case names. They're
	// the discriminator strings ctrl writes into the CH event_kind column,
	// the labels on dedupe-cache keys, and the metrics labels.
	eventKindRunning    = "running"
	eventKindTerminated = "terminated"
	eventKindWaiting    = "waiting"

	// fingerprintMessageMax bounds the message bytes used in the
	// event_fingerprint hash. Long stack traces make every retry of the
	// "same" failure look unique. 200 bytes is enough to capture a panic
	// header or kubelet error string while collapsing noisier suffixes.
	fingerprintMessageMax = 200
)

// reportInstanceEvents walks the pod's container statuses, builds the set of
// events not yet seen for this (pod_uid, container_name, restart_count,
// state) tuple, and ships them in a single batched RPC. Best-effort: errors
// are logged and surfaced as metrics but do not fail the caller.
func (c *Controller) reportInstanceEvents(ctx context.Context, pod *corev1.Pod) {
	if c.eventDedup == nil {
		return // not configured (tests or environments without ctrl-CH wiring)
	}

	candidates := scanInstanceEvents(pod)
	if len(candidates) == 0 {
		return
	}

	fresh := make([]*ctrlv1.InstanceEvent, 0, len(candidates))
	for _, ev := range candidates {
		key := dedupKey(ev)
		if _, hit := c.eventDedup.Get(ctx, key); hit == cache.Hit {
			metrics.InstanceEventsDedupDroppedTotal.WithLabelValues(eventKindOf(ev)).Inc()
			continue
		}
		c.eventDedup.Set(ctx, key, struct{}{})
		fresh = append(fresh, ev)
	}
	if len(fresh) == 0 {
		return
	}

	req := &ctrlv1.ReportInstanceEventsRequest{
		Events: fresh,
		// Same RegionKey-on-the-body convention every other ClusterService
		// RPC follows. Without this, ctrl can't map the event to the right
		// region row and rejects the call with InvalidArgument.
		Region: c.regionKey(),
	}
	// Wrap the RPC in the controller's circuit breaker. Same backend as
	// reportDeploymentStatus, so a ctrl outage trips one breaker for both
	// call sites and we fail fast everywhere.
	if _, err := c.cb.Do(ctx, func(innerCtx context.Context) (any, error) {
		return c.cluster.ReportInstanceEvents(innerCtx, req)
	}); err != nil {
		metrics.InstanceEventsReportFailuresTotal.Inc()
		logger.Error("instance events: report rpc failed",
			"error", err.Error(),
			"pod", pod.Name,
			"events", len(fresh),
		)
		// Roll back the dedup entries on transport failure so the next
		// pod-watch tick will re-emit. The cache TTL would eventually
		// recover this anyway, but explicit rollback keeps recovery
		// time bounded by the watch interval, not the cache stale time.
		for _, ev := range fresh {
			c.eventDedup.Remove(ctx, dedupKey(ev))
		}
		return
	}

	for _, ev := range fresh {
		metrics.InstanceEventsEmittedTotal.WithLabelValues(eventKindOf(ev), reasonOf(ev)).Inc()
	}
}

// scanInstanceEvents inspects a pod's container statuses and returns one
// event per (container, life, state) tuple we should report. Pure function,
// no I/O.
//
// For each ContainerStatus and InitContainerStatus we look at:
//   - State.Running: emit a Running event for the current life. Idempotent
//     across pod-watch ticks via the dedupe cache.
//   - State.Terminated: the current container life ended at restart_count.
//   - LastTerminationState.Terminated: the previous life ended at
//     (restart_count - 1) and the container has since restarted. Skipped
//     when restart_count == 0 because there is no prior life to describe.
//   - State.Waiting with reason=CrashLoopBackOff: kubelet has put the
//     container in a backoff window. Emit one event per restart_count
//     value so the dashboard can render "kubelet is throttling" alongside
//     the underlying exit.
func scanInstanceEvents(pod *corev1.Pod) []*ctrlv1.InstanceEvent {
	if pod == nil {
		return nil
	}
	tenant := extractTenantContext(pod)
	if tenant.deploymentID == "" {
		// Non-krane pods (e.g. system pods that somehow match the watch
		// selector) don't get instance events — there's no deployment for
		// the dashboard to attribute them to.
		return nil
	}

	statuses := append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...)
	statuses = append(statuses, pod.Status.ContainerStatuses...)

	out := make([]*ctrlv1.InstanceEvent, 0, len(statuses))
	for _, cs := range statuses {
		if r := cs.State.Running; r != nil {
			out = append(out, buildRunningEvent(pod, tenant, cs, r.StartedAt.UnixMilli()))
		}
		if t := cs.State.Terminated; t != nil {
			out = append(out, buildTerminatedEvent(pod, tenant, cs, cs.RestartCount, t))
		}
		if cs.RestartCount > 0 && cs.LastTerminationState.Terminated != nil {
			out = append(out, buildTerminatedEvent(pod, tenant, cs, cs.RestartCount-1, cs.LastTerminationState.Terminated))
		}
		if w := cs.State.Waiting; w != nil && w.Reason == "CrashLoopBackOff" {
			out = append(out, buildWaitingEvent(pod, tenant, cs, w))
		}
	}
	return out
}

// tenantContext bundles the workspace/project/app/environment/deployment
// IDs we extract once per pod so we don't re-parse labels per container
// status.
type tenantContext struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	deploymentID  string
}

func extractTenantContext(pod *corev1.Pod) tenantContext {
	l := pod.Labels
	return tenantContext{
		workspaceID:   l[labels.LabelKeyWorkspaceID],
		projectID:     l[labels.LabelKeyProjectID],
		appID:         l[labels.LabelKeyAppID],
		environmentID: l[labels.LabelKeyEnvironmentID],
		deploymentID:  l[labels.LabelKeyDeploymentID],
	}
}

// newEvent fills in the identity, tenant, time, and attribute fields shared
// by every state. Per-state builders set the oneof case and per-state
// fingerprint on top of the returned struct.
func newEvent(
	pod *corev1.Pod,
	tenant tenantContext,
	cs corev1.ContainerStatus,
	restartCount int32,
	when int64,
) *ctrlv1.InstanceEvent {
	return &ctrlv1.InstanceEvent{
		PodUid:        string(pod.UID),
		PodName:       pod.Name,
		NodeName:      pod.Spec.NodeName,
		ContainerName: cs.Name,
		ContainerId:   cs.ContainerID,
		RestartCount:  restartCount,
		WorkspaceId:   tenant.workspaceID,
		ProjectId:     tenant.projectID,
		AppId:         tenant.appID,
		EnvironmentId: tenant.environmentID,
		DeploymentId:  tenant.deploymentID,
		Time:          when,
		Attributes:    extractAttributes(pod, cs),
	}
}

func buildRunningEvent(pod *corev1.Pod, tenant tenantContext, cs corev1.ContainerStatus, startedAt int64) *ctrlv1.InstanceEvent {
	if startedAt <= 0 {
		// Kubelet hasn't filled StartedAt yet; fall back to "now" so the
		// row partitions sensibly rather than landing in the epoch bucket.
		startedAt = time.Now().UnixMilli()
	}
	ev := newEvent(pod, tenant, cs, cs.RestartCount, startedAt)
	ev.State = &ctrlv1.InstanceEvent_Running{Running: &ctrlv1.Running{}}
	ev.EventFingerprint = fingerprint(cs.ImageID, 0, eventKindRunning, "")
	return ev
}

func buildTerminatedEvent(
	pod *corev1.Pod,
	tenant tenantContext,
	cs corev1.ContainerStatus,
	restartCount int32,
	t *corev1.ContainerStateTerminated,
) *ctrlv1.InstanceEvent {
	when := t.FinishedAt.UnixMilli()
	if when <= 0 {
		// Kubelet sometimes lags on FinishedAt; same fallback as Running.
		when = time.Now().UnixMilli()
	}
	ev := newEvent(pod, tenant, cs, restartCount, when)
	ev.ContainerId = t.ContainerID
	ev.State = &ctrlv1.InstanceEvent_Terminated{Terminated: &ctrlv1.Terminated{
		ExitCode: t.ExitCode,
		Signal:   t.Signal,
		Reason:   t.Reason,
		Message:  t.Message,
	}}
	ev.EventFingerprint = fingerprint(cs.ImageID, t.ExitCode, t.Reason, t.Message)
	return ev
}

func buildWaitingEvent(pod *corev1.Pod, tenant tenantContext, cs corev1.ContainerStatus, w *corev1.ContainerStateWaiting) *ctrlv1.InstanceEvent {
	ev := newEvent(pod, tenant, cs, cs.RestartCount, time.Now().UnixMilli())
	ev.State = &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{
		Reason:  w.Reason,
		Message: w.Message,
	}}
	// Including the most recent exit code/reason from LastTerminationState
	// would make every new exit "look different" even when the underlying
	// loop is the same. Using just (image_id, waiting reason, message)
	// keeps the fingerprint stable across the loop.
	ev.EventFingerprint = fingerprint(cs.ImageID, 0, w.Reason, w.Message)
	return ev
}

// eventKindOf returns the discriminator string for an event's oneof case.
// Used for metrics labels and the CH dedupe key. Returns "" if no state is
// set, which is a programming error — the caller would have already failed
// the dedupe lookup.
func eventKindOf(ev *ctrlv1.InstanceEvent) string {
	switch ev.GetState().(type) {
	case *ctrlv1.InstanceEvent_Running:
		return eventKindRunning
	case *ctrlv1.InstanceEvent_Terminated:
		return eventKindTerminated
	case *ctrlv1.InstanceEvent_Waiting:
		return eventKindWaiting
	default:
		return ""
	}
}

// reasonOf surfaces the kubelet-supplied label for the metric. Empty for
// running events (no reason concept).
func reasonOf(ev *ctrlv1.InstanceEvent) string {
	switch s := ev.GetState().(type) {
	case *ctrlv1.InstanceEvent_Terminated:
		return s.Terminated.GetReason()
	case *ctrlv1.InstanceEvent_Waiting:
		return s.Waiting.GetReason()
	default:
		return ""
	}
}

// extractAttributes pulls the small set of debug-context fields the
// dashboard renders alongside the event row: image identity, resource
// limits/requests, build_id. Empty values are omitted so the CH map stays
// compact and the dashboard can `if attrs.image` cleanly.
//
// Resource numbers come from the pod spec, not the container status —
// kubelet's ContainerStatus doesn't carry the limits, only the live
// process state. We match by container name to the corresponding spec
// entry; if the container isn't found in the spec (shouldn't happen for
// krane-managed pods, but possible for sidecar weirdness) we just skip
// the resource keys.
func extractAttributes(pod *corev1.Pod, cs corev1.ContainerStatus) map[string]string {
	attrs := map[string]string{}
	setIfNotEmpty := func(key, value string) {
		if value != "" {
			attrs[key] = value
		}
	}
	setIfNotEmpty("image", cs.Image)
	setIfNotEmpty("image_id", cs.ImageID)
	setIfNotEmpty("build_id", pod.Labels[labels.LabelKeyBuildID])

	if spec := findContainerSpec(pod, cs.Name); spec != nil {
		if v := spec.Resources.Limits.Cpu(); !v.IsZero() {
			attrs["cpu_limit_millicores"] = strconv.FormatInt(v.MilliValue(), 10)
		}
		if v := spec.Resources.Limits.Memory(); !v.IsZero() {
			attrs["memory_limit_mib"] = strconv.FormatInt(v.Value()/(1024*1024), 10)
		}
		if v := spec.Resources.Requests.Cpu(); !v.IsZero() {
			attrs["cpu_request_millicores"] = strconv.FormatInt(v.MilliValue(), 10)
		}
		if v := spec.Resources.Requests.Memory(); !v.IsZero() {
			attrs["memory_request_mib"] = strconv.FormatInt(v.Value()/(1024*1024), 10)
		}
	}
	return attrs
}

// findContainerSpec returns the pod's container spec matching the given
// name, scanning both regular and init containers. Returns nil when the
// container isn't part of the spec (which would be unusual for a status
// row whose name was just produced from the same spec).
func findContainerSpec(pod *corev1.Pod, name string) *corev1.Container {
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == name {
			return &pod.Spec.Containers[i]
		}
	}
	for i := range pod.Spec.InitContainers {
		if pod.Spec.InitContainers[i].Name == name {
			return &pod.Spec.InitContainers[i]
		}
	}
	return nil
}

// fingerprint hashes the inputs that should make two failures "the same
// incident" for grouping purposes. Trimming message keeps stack traces from
// making every retry look unique.
func fingerprint(imageID string, exitCode int32, reason, message string) string {
	if len(message) > fingerprintMessageMax {
		message = message[:fingerprintMessageMax]
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s|%s", imageID, exitCode, reason, message)))
	return hex.EncodeToString(h[:])
}

// dedupKey is the in-memory dedupe identity for an event. Mirrors the
// ClickHouse insert dedupe constraint.
func dedupKey(ev *ctrlv1.InstanceEvent) string {
	return strings.Join([]string{
		ev.GetPodUid(),
		ev.GetContainerName(),
		strconv.FormatInt(int64(ev.GetRestartCount()), 10),
		eventKindOf(ev),
	}, "|")
}
