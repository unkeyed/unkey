package reconciler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetDeploymentState_PodStatusMapping(t *testing.T) {
	tests := []struct {
		name           string
		phase          corev1.PodPhase
		expectedStatus ctrlv1.UpdateDeploymentStateRequest_Update_Instance_Status
	}{
		{
			name:           "pending pod",
			phase:          corev1.PodPending,
			expectedStatus: ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_PENDING,
		},
		{
			name:           "running pod",
			phase:          corev1.PodRunning,
			expectedStatus: ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_RUNNING,
		},
		{
			name:           "failed pod",
			phase:          corev1.PodFailed,
			expectedStatus: ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_FAILED,
		},
		{
			name:           "succeeded pod",
			phase:          corev1.PodSucceeded,
			expectedStatus: ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED,
		},
		{
			name:           "unknown pod",
			phase:          corev1.PodUnknown,
			expectedStatus: ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED,
		},
		{
			name:           "default case",
			phase:          corev1.PodPhase("SomeUnknownPhase"),
			expectedStatus: ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			rs := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rs",
					Namespace: "test-namespace",
				},
				Spec: appsv1.ReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
			}

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
					Labels:    map[string]string{"app": "test"},
				},
				Status: corev1.PodStatus{
					Phase: tt.phase,
					PodIP: "10.0.0.1",
				},
			}

			client := fake.NewSimpleClientset(rs, pod)
			r := &Reconciler{
				clientSet: client,
				cluster:   &MockClusterClient{},
				cb:        circuitbreaker.New[any]("test"),
				logger:    logging.NewNoop(),
				region:    "test-region",
			}

			state, err := r.getDeploymentState(ctx, rs)
			require.NoError(t, err)

			update := state.GetUpdate()
			require.NotNil(t, update)
			require.Len(t, update.GetInstances(), 1)
			require.Equal(t, tt.expectedStatus, update.GetInstances()[0].GetStatus())
		})
	}
}

func TestGetDeploymentState_AddressFormatting(t *testing.T) {
	ctx := context.Background()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rs",
			Namespace: "my-namespace",
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "my-namespace",
			Labels:    map[string]string{"app": "test"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
		},
	}

	client := fake.NewSimpleClientset(rs, pod)
	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state, err := r.getDeploymentState(ctx, rs)
	require.NoError(t, err)

	update := state.GetUpdate()
	require.NotNil(t, update)
	require.Len(t, update.GetInstances(), 1)
	require.Equal(t, "10-0-0-1.my-namespace.pod.cluster.local", update.GetInstances()[0].GetAddress())
}

func TestGetDeploymentState_ResourceExtraction(t *testing.T) {
	ctx := context.Background()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rs",
			Namespace: "test-namespace",
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			Labels:    map[string]string{"app": "test"},
		},
		Spec: corev1.PodSpec{
			Resources: &corev1.ResourceRequirements{
				// nolint:exhaustive
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
		},
	}

	client := fake.NewSimpleClientset(rs, pod)
	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state, err := r.getDeploymentState(ctx, rs)
	require.NoError(t, err)

	update := state.GetUpdate()
	require.NotNil(t, update)
	require.Len(t, update.GetInstances(), 1)
	require.Equal(t, int64(500), update.GetInstances()[0].GetCpuMillicores())
	require.Equal(t, int64(256), update.GetInstances()[0].GetMemoryMib())
}

func TestGetDeploymentState_MultiplePods(t *testing.T) {
	ctx := context.Background()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rs",
			Namespace: "test-namespace",
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
	}

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: "test-namespace",
			Labels:    map[string]string{"app": "test"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-2",
			Namespace: "test-namespace",
			Labels:    map[string]string{"app": "test"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			PodIP: "10.0.0.2",
		},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-3",
			Namespace: "test-namespace",
			Labels:    map[string]string{"app": "test"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			PodIP: "10.0.0.3",
		},
	}

	client := fake.NewSimpleClientset(rs, pod1, pod2, pod3)
	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state, err := r.getDeploymentState(ctx, rs)
	require.NoError(t, err)

	update := state.GetUpdate()
	require.NotNil(t, update)
	require.Len(t, update.GetInstances(), 3)

	podNames := make(map[string]bool)
	for _, instance := range update.GetInstances() {
		podNames[instance.GetK8SName()] = true
	}
	require.True(t, podNames["test-pod-1"])
	require.True(t, podNames["test-pod-2"])
	require.True(t, podNames["test-pod-3"])

	require.Equal(t, "test-rs", update.GetK8SName())
}

func TestGetDeploymentState_EmptyPodIP(t *testing.T) {
	ctx := context.Background()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rs",
			Namespace: "test-namespace",
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
	}

	podWithIP := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-with-ip",
			Namespace: "test-namespace",
			Labels:    map[string]string{"app": "test"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
		},
	}

	podWithoutIP := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-without-ip",
			Namespace: "test-namespace",
			Labels:    map[string]string{"app": "test"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			PodIP: "",
		},
	}

	client := fake.NewSimpleClientset(rs, podWithIP, podWithoutIP)
	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state, err := r.getDeploymentState(ctx, rs)
	require.NoError(t, err)

	update := state.GetUpdate()
	require.NotNil(t, update)
	require.Len(t, update.GetInstances(), 1, "pods with empty PodIP should be skipped")
	require.Equal(t, "pod-with-ip", update.GetInstances()[0].GetK8SName())
	require.Equal(t, "10-0-0-1.test-namespace.pod.cluster.local", update.GetInstances()[0].GetAddress())
}
