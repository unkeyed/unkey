package deployment

import (
	"testing"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// scanInstanceEvents is the pure decision layer for capturing pod-level
// container failures. The dedup LRU and RPC layer sit on top; testing them
// requires a full Controller wiring. These cases focus on which kubelet
// states should map to which oneof case / restart_count combinations.
func TestScanInstanceEvents(t *testing.T) {
	t.Parallel()

	makePod := func(uid string, statuses []corev1.ContainerStatus) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-" + uid,
				Namespace: "default",
				UID:       types.UID(uid),
				Labels: map[string]string{
					labels.LabelKeyWorkspaceID:   "ws_1",
					labels.LabelKeyProjectID:     "proj_1",
					labels.LabelKeyAppID:         "app_1",
					labels.LabelKeyEnvironmentID: "env_1",
					labels.LabelKeyDeploymentID:  "dep_1",
				},
			},
			Spec:   corev1.PodSpec{NodeName: "node-1"},
			Status: corev1.PodStatus{ContainerStatuses: statuses},
		}
	}

	terminated := func(reason string, exitCode int32, finishedAt time.Time) *corev1.ContainerStateTerminated {
		return &corev1.ContainerStateTerminated{
			ExitCode:   exitCode,
			Reason:     reason,
			FinishedAt: metav1.NewTime(finishedAt),
			StartedAt:  metav1.NewTime(finishedAt.Add(-30 * time.Second)),
		}
	}

	t.Run("running pod emits a single Running event", func(t *testing.T) {
		t.Parallel()
		startedAt := time.UnixMilli(1700000000000)
		pod := makePod("uid-1", []corev1.ContainerStatus{{
			Name:         "app",
			RestartCount: 0,
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{
				StartedAt: metav1.NewTime(startedAt),
			}},
		}})
		got := scanInstanceEvents(pod)
		if len(got) != 1 {
			t.Fatalf("expected 1 event, got %d", len(got))
		}
		ev := got[0]
		if _, ok := ev.GetState().(*ctrlv1.InstanceEvent_Running); !ok {
			t.Errorf("state: got %T want *InstanceEvent_Running", ev.GetState())
		}
		if ev.GetTime() != startedAt.UnixMilli() {
			t.Errorf("time: got %d want %d", ev.GetTime(), startedAt.UnixMilli())
		}
		if ev.GetTerminated() != nil || ev.GetWaiting() != nil {
			t.Errorf("Running event must not carry terminated/waiting metadata")
		}
	})

	t.Run("currently terminated container produces one event keyed on current restart_count", func(t *testing.T) {
		t.Parallel()
		now := time.UnixMilli(1700000000000)
		pod := makePod("uid-2", []corev1.ContainerStatus{{
			Name:         "app",
			RestartCount: 0,
			State:        corev1.ContainerState{Terminated: terminated("Error", 1, now)},
		}})
		got := scanInstanceEvents(pod)
		if len(got) != 1 {
			t.Fatalf("expected 1 event, got %d", len(got))
		}
		ev := got[0]
		term := ev.GetTerminated()
		if term == nil {
			t.Fatalf("expected Terminated state, got %T", ev.GetState())
		}
		if ev.GetRestartCount() != 0 {
			t.Errorf("restart_count: got %d want 0", ev.GetRestartCount())
		}
		if term.GetExitCode() != 1 || term.GetReason() != "Error" {
			t.Errorf("exit/reason mismatch: got %d/%q", term.GetExitCode(), term.GetReason())
		}
	})

	t.Run("running container with last termination state keys the prior life", func(t *testing.T) {
		t.Parallel()
		// Container has restarted 3 times; LastTerminationState describes
		// the death between life #2 and life #3, so the Terminated event
		// restart_count is 2 (one less than the live count). The current
		// life #3 also produces a Running event at restart_count=3.
		now := time.UnixMilli(1700000000000)
		pod := makePod("uid-3", []corev1.ContainerStatus{{
			Name:                 "app",
			RestartCount:         3,
			State:                corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(now)}},
			LastTerminationState: corev1.ContainerState{Terminated: terminated("OOMKilled", 137, now)},
		}})
		got := scanInstanceEvents(pod)
		if len(got) != 2 {
			t.Fatalf("expected 2 events (Running for current life, Terminated for prior), got %d", len(got))
		}
		var term, running *ctrlv1.InstanceEvent
		for _, ev := range got {
			switch ev.GetState().(type) {
			case *ctrlv1.InstanceEvent_Terminated:
				term = ev
			case *ctrlv1.InstanceEvent_Running:
				running = ev
			}
		}
		if term == nil || running == nil {
			t.Fatalf("missing kinds: running=%v terminated=%v", running != nil, term != nil)
		}
		if term.GetRestartCount() != 2 {
			t.Errorf("term restart_count: got %d want 2", term.GetRestartCount())
		}
		if td := term.GetTerminated(); td.GetReason() != "OOMKilled" || td.GetExitCode() != 137 {
			t.Errorf("oom mismatch: %s / %d", td.GetReason(), td.GetExitCode())
		}
		if running.GetRestartCount() != 3 {
			t.Errorf("running restart_count: got %d want 3", running.GetRestartCount())
		}
	})

	t.Run("crashloop backoff emits a separate Waiting event", func(t *testing.T) {
		t.Parallel()
		now := time.UnixMilli(1700000000000)
		pod := makePod("uid-4", []corev1.ContainerStatus{{
			Name:         "app",
			RestartCount: 5,
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{
				Reason:  "CrashLoopBackOff",
				Message: "back-off 5m0s restarting failed container",
			}},
			LastTerminationState: corev1.ContainerState{Terminated: terminated("Error", 1, now)},
		}})
		got := scanInstanceEvents(pod)
		// One Terminated event (for life #4 = restart_count - 1) plus
		// one Waiting event for the current waiting state.
		if len(got) != 2 {
			t.Fatalf("expected 2 events, got %d", len(got))
		}
		var sawTerm, sawWaiting bool
		for _, ev := range got {
			switch s := ev.GetState().(type) {
			case *ctrlv1.InstanceEvent_Terminated:
				sawTerm = true
				if ev.GetRestartCount() != 4 {
					t.Errorf("term restart_count: got %d want 4", ev.GetRestartCount())
				}
			case *ctrlv1.InstanceEvent_Waiting:
				sawWaiting = true
				if ev.GetRestartCount() != 5 {
					t.Errorf("waiting restart_count: got %d want 5", ev.GetRestartCount())
				}
				if s.Waiting.GetReason() != "CrashLoopBackOff" {
					t.Errorf("waiting reason: got %q", s.Waiting.GetReason())
				}
			}
		}
		if !sawTerm || !sawWaiting {
			t.Errorf("missing kinds: term=%v waiting=%v", sawTerm, sawWaiting)
		}
	})

	t.Run("restart_count zero does not synthesize a phantom prior life", func(t *testing.T) {
		t.Parallel()
		// First-life container that has never restarted. Even if some
		// other field looks populated, we don't emit a Terminated event for
		// "lifetime -1" — there is no prior life. The current Running life
		// still produces a Running event.
		now := time.UnixMilli(1700000000000)
		pod := makePod("uid-5", []corev1.ContainerStatus{{
			Name:                 "app",
			RestartCount:         0,
			State:                corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(now)}},
			LastTerminationState: corev1.ContainerState{Terminated: terminated("Error", 1, now)},
		}})
		got := scanInstanceEvents(pod)
		if len(got) != 1 {
			t.Fatalf("expected 1 event (Running, no phantom prior life), got %d", len(got))
		}
		if _, ok := got[0].GetState().(*ctrlv1.InstanceEvent_Running); !ok {
			t.Errorf("expected Running event, got %T", got[0].GetState())
		}
	})

	t.Run("non-krane pods (no deployment_id label) are skipped", func(t *testing.T) {
		t.Parallel()
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "system", UID: "x", Labels: map[string]string{}},
			Spec:       corev1.PodSpec{NodeName: "node-1"},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
				Name:         "app",
				RestartCount: 0,
				State:        corev1.ContainerState{Terminated: terminated("Error", 1, time.UnixMilli(1700000000000))},
			}}},
		}
		got := scanInstanceEvents(pod)
		if len(got) != 0 {
			t.Fatalf("expected non-krane pod to be skipped, got %d events", len(got))
		}
	})

	t.Run("init containers are scanned alongside main containers", func(t *testing.T) {
		t.Parallel()
		now := time.UnixMilli(1700000000000)
		pod := makePod("uid-6", nil)
		pod.Status.InitContainerStatuses = []corev1.ContainerStatus{{
			Name:         "migrate",
			RestartCount: 0,
			State:        corev1.ContainerState{Terminated: terminated("ContainerCannotRun", 127, now)},
		}}
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
			Name:         "app",
			RestartCount: 0,
			State:        corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "PodInitializing"}},
		}}
		got := scanInstanceEvents(pod)
		if len(got) != 1 {
			t.Fatalf("expected 1 event from init container, got %d", len(got))
		}
		if got[0].GetContainerName() != "migrate" {
			t.Errorf("expected init container event, got name %q", got[0].GetContainerName())
		}
	})
}

func TestFingerprintIsStable(t *testing.T) {
	t.Parallel()
	a := fingerprint("img@sha256:abc", 137, "OOMKilled", "killed by kernel")
	b := fingerprint("img@sha256:abc", 137, "OOMKilled", "killed by kernel")
	if a != b {
		t.Fatalf("fingerprint not stable: %q vs %q", a, b)
	}
	c := fingerprint("img@sha256:abc", 1, "Error", "killed by kernel")
	if a == c {
		t.Fatalf("fingerprint should differ for different exit codes")
	}
}

func TestExtractAttributesPopulatesKnownKeys(t *testing.T) {
	t.Parallel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-attrs",
			UID:  "uid-attrs",
			Labels: map[string]string{
				labels.LabelKeyWorkspaceID:  "ws_1",
				labels.LabelKeyProjectID:    "proj_1",
				labels.LabelKeyDeploymentID: "dep_1",
				labels.LabelKeyBuildID:      "bld_42",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{{
				Name: "app",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("64Mi"),
					},
				},
			}},
		},
	}
	cs := corev1.ContainerStatus{
		Name:    "app",
		Image:   "ghcr.io/test/app:v1",
		ImageID: "ghcr.io/test/app@sha256:abc123",
	}
	attrs := extractAttributes(pod, cs)
	cases := map[string]string{
		"image":                  "ghcr.io/test/app:v1",
		"image_id":               "ghcr.io/test/app@sha256:abc123",
		"build_id":               "bld_42",
		"cpu_limit_millicores":   "500",
		"memory_limit_mib":       "256",
		"cpu_request_millicores": "100",
		"memory_request_mib":     "64",
	}
	for key, want := range cases {
		if got := attrs[key]; got != want {
			t.Errorf("attrs[%q]: got %q want %q", key, got, want)
		}
	}
}

func TestExtractAttributesOmitsEmptyValues(t *testing.T) {
	t.Parallel()
	// Container with no resource limits set, no image_id yet, no build label —
	// each missing field should be absent from the map rather than mapped to "".
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "pod-bare",
			UID:    "uid-bare",
			Labels: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
	}
	cs := corev1.ContainerStatus{Name: "app", Image: "test:latest"}
	attrs := extractAttributes(pod, cs)
	if _, ok := attrs["image_id"]; ok {
		t.Errorf("expected image_id to be omitted when empty, got %q", attrs["image_id"])
	}
	if _, ok := attrs["memory_limit_mib"]; ok {
		t.Errorf("expected memory_limit_mib to be omitted when no limit set")
	}
	if attrs["image"] != "test:latest" {
		t.Errorf("image should still be set when present: %q", attrs["image"])
	}
}

func TestDedupKeyIncludesState(t *testing.T) {
	t.Parallel()
	term := &ctrlv1.InstanceEvent{
		PodUid:        "uid-1",
		ContainerName: "app",
		RestartCount:  3,
		State:         &ctrlv1.InstanceEvent_Terminated{Terminated: &ctrlv1.Terminated{}},
	}
	waiting := &ctrlv1.InstanceEvent{
		PodUid:        "uid-1",
		ContainerName: "app",
		RestartCount:  3,
		State:         &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{Reason: "CrashLoopBackOff"}},
	}
	if dedupKey(term) == dedupKey(waiting) {
		t.Fatalf("dedup key should distinguish state: both produced %q", dedupKey(term))
	}
}
