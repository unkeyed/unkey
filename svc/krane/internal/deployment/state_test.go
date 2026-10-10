package deployment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBuildDeploymentStatus_PodStatuses(t *testing.T) {
	rsSelector := map[string]string{"deployment_id": "dep_abc"}
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
				Labels:    rsSelector,
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

	client := fake.NewSimpleClientset(
		&runningReady, &runningUnready, &runningNoCondition, &pending, &terminating, &otherPort, &failed, &succeeded,
	)
	ctrl := New(Config{
		ClientSet:     client,
		DynamicClient: fakedynamic.NewSimpleDynamicClient(runtime.NewScheme()),
		Cluster:       &testutil.MockClusterClient{},
		Region:        "local",
		Fingerprints:  cache.NewNoopCache[string, string](),
	})

	status, err := ctrl.buildDeploymentStatus(context.Background(), w)
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
