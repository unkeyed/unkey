package deployment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBuildDeploymentStatus_PodStatuses(t *testing.T) {
	rsSelector := map[string]string{"deployment_id": "dep_abc"}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-1",
			Namespace: "ns-1",
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: rsSelector},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Ports: []corev1.ContainerPort{{ContainerPort: 8080}},
					}},
				},
			},
		},
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

	failed := podBase("pod-failed")
	failed.Status.Phase = corev1.PodFailed

	client := fake.NewSimpleClientset(
		&runningReady, &runningUnready, &runningNoCondition, &pending, &failed,
	)
	ctrl := New(Config{
		ClientSet:     client,
		DynamicClient: fakedynamic.NewSimpleDynamicClient(runtime.NewScheme()),
		Cluster:       &testutil.MockClusterClient{},
		Region:        "local",
		Fingerprints:  cache.NewNoopCache[string, string](),
	})

	status, err := ctrl.buildDeploymentStatus(context.Background(), rs)
	require.NoError(t, err)

	byName := map[string]ctrlv1.ReportDeploymentStatusRequest_Update_Instance_Status{}
	for _, inst := range status.GetUpdate().GetInstances() {
		byName[inst.GetK8SName()] = inst.GetStatus()
	}

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
	require.Equal(t,
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_FAILED,
		byName["pod-failed"],
	)
}
