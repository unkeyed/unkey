package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildDeploymentStatus_PodStatuses(t *testing.T) {
	rsSelector := map[string]string{labels.LabelKeyDeploymentID: "dep_abc"}
	w := workload{
		namespace:    "ns-1",
		k8sName:      "rs-1",
		deploymentID: "dep_abc",
		selector:     &metav1.LabelSelector{MatchLabels: rsSelector},
	}

	podBase := func(name string) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns-1",
				Labels:    labels.New().ManagedByKrane().ComponentDeployment().DeploymentID("dep_abc"),
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}
	}

	runningReady := podBase("pod-ready")
	runningReady.Status.Phase = corev1.PodRunning
	runningReady.Status.Conditions = []corev1.PodCondition{{
		Type:   corev1.ContainersReady,
		Status: corev1.ConditionTrue,
	}}

	runningUnready := podBase("pod-unready")
	runningUnready.Status.Phase = corev1.PodRunning
	runningUnready.Status.Conditions = []corev1.PodCondition{{
		Type:   corev1.ContainersReady,
		Status: corev1.ConditionFalse,
	}}

	runningNoCondition := podBase("pod-no-condition")
	runningNoCondition.Status.Phase = corev1.PodRunning

	pending := podBase("pod-pending")
	pending.Status.Phase = corev1.PodPending

	terminating := podBase("pod-terminating")
	terminating.Status.Phase = corev1.PodRunning
	terminating.DeletionTimestamp = new(metav1.Now())
	terminating.Finalizers = []string{"test/keep"}

	otherPort := podBase("pod-other-port")
	otherPort.Status.Phase = corev1.PodRunning
	otherPort.Status.PodIP = "10.0.0.2"
	otherPort.Spec.Containers = []corev1.Container{{
		Name:  "app",
		Ports: []corev1.ContainerPort{{ContainerPort: 3000}},
	}}

	failed := podBase("pod-failed")
	failed.Status.Phase = corev1.PodFailed
	succeeded := podBase("pod-succeeded")
	succeeded.Status.Phase = corev1.PodSucceeded

	ctrl := cachedController(t,
		&runningReady, &runningUnready, &runningNoCondition, &pending, &terminating, &otherPort, &failed, &succeeded,
	)

	status, err := ctrl.buildDeploymentStatus(w)
	require.NoError(t, err)

	byName := map[string]ctrlv1.ReportDeploymentStatusRequest_Update_Instance_Status{}
	addressesByName := map[string]string{}
	for _, inst := range status.GetUpdate().GetInstances() {
		byName[inst.GetK8SName()] = inst.GetStatus()
		addressesByName[inst.GetK8SName()] = inst.GetAddress()
	}
	require.Equal(t, "10.0.0.1:8080", addressesByName["pod-ready"])
	require.Equal(t, "10.0.0.2:3000", addressesByName["pod-other-port"], "address must use the pod's own port")
	require.Equal(t, "rs-1", status.GetUpdate().GetK8SName())

	require.Equal(t,
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING,
		byName["pod-ready"],
		"ContainersReady=True should map to RUNNING",
	)
	require.Equal(t,
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING,
		byName["pod-unready"],
		"ContainersReady=False should map to PENDING, not FAILED",
	)
	require.Equal(t,
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING,
		byName["pod-no-condition"],
		"PodRunning with no ContainersReady condition should map to PENDING, not RUNNING",
	)
	require.Equal(t,
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING,
		byName["pod-pending"],
	)
	require.NotContains(t, byName, "pod-terminating", "terminating pods must not receive traffic")
	require.NotContains(t, byName, "pod-failed", "failed pods must not remain active instances")
	require.NotContains(t, byName, "pod-succeeded", "succeeded pods must not remain active instances")
}

func TestRoutablePods(t *testing.T) {
	pod := func(name string, drains, ready, terminating bool) *corev1.Pod {
		p := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}
		if drains {
			p.Spec.Containers[0].Lifecycle = &corev1.Lifecycle{PreStop: preStopHandler("")}
		}
		if ready {
			p.Status.Conditions = []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}}
		}
		if terminating {
			p.DeletionTimestamp = new(metav1.Now())
		}
		return p
	}
	names := func(pods []*corev1.Pod) []string {
		out := []string{}
		for _, p := range pods {
			out = append(out, p.Name)
		}
		return out
	}

	for _, tt := range []struct {
		name string
		pods []*corev1.Pod
		want []string
	}{
		{"legacy pods stay routable until replacements exist",
			[]*corev1.Pod{pod("old-1", false, true, false), pod("old-2", false, true, false)},
			[]string{"old-1", "old-2"}},
		{"legacy pods stay routable while replacements start",
			[]*corev1.Pod{pod("old-1", false, true, false), pod("old-2", false, true, false), pod("new-1", true, true, false), pod("new-2", true, false, false)},
			[]string{"new-1", "new-2", "old-1", "old-2"}},
		{"legacy pods leave routing once enough replacements are ready",
			[]*corev1.Pod{pod("old-1", false, true, false), pod("old-2", false, true, false), pod("new-1", true, true, false), pod("new-2", true, true, false)},
			[]string{"new-1", "new-2"}},
		{"single replica",
			[]*corev1.Pod{pod("old-1", false, true, false), pod("new-1", true, true, false)},
			[]string{"new-1"}},
		{"terminating pods leave routing while another pod is ready",
			[]*corev1.Pod{pod("new-1", true, true, false), pod("new-2", true, true, true)},
			[]string{"new-1"}},
		{"an evicted single replica serves until its replacement is ready",
			[]*corev1.Pod{pod("evicted", true, true, true), pod("replacement", true, false, false)},
			[]string{"replacement", "evicted"}},
		{"an evicted single replica leaves routing once its replacement is ready",
			[]*corev1.Pod{pod("evicted", true, true, true), pod("replacement", true, true, false)},
			[]string{"replacement"}},
		{"terminating pods that cannot drain are never routable",
			[]*corev1.Pod{pod("evicted", false, true, true), pod("replacement", true, false, false)},
			[]string{"replacement"}},
		{"terminating pods that stopped being ready are not routable",
			[]*corev1.Pod{pod("evicted", true, false, true)},
			[]string{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, names(routablePods(tt.pods)))
		})
	}
}
