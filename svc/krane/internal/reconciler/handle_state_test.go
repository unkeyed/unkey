package reconciler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestHandleState_DeploymentApply(t *testing.T) {
	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)

	var appliedRS *appsv1.ReplicaSet
	client.PrependReactor("patch", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var rs appsv1.ReplicaSet
		if err := json.Unmarshal(patchAction.GetPatch(), &rs); err != nil {
			return true, nil, err
		}

		appliedRS = &rs
		rs.Namespace = patchAction.GetNamespace()
		return true, &rs, nil
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: &ctrlv1.ApplyDeployment{
						WorkspaceId:   "ws_123",
						ProjectId:     "prj_123",
						EnvironmentId: "env_123",
						DeploymentId:  "dep_123",
						K8SNamespace:  "test-namespace",
						K8SName:       "test-deployment",
						Image:         "nginx:1.19",
						Replicas:      3,
						CpuMillicores: 100,
						MemoryMib:     128,
						BuildId:       ptr.P("build_123"),
					},
				},
			},
		},
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)

	require.NotNil(t, appliedRS, "ApplyDeployment should have been called, creating a ReplicaSet")
	require.Equal(t, "test-deployment", appliedRS.Name)
}

func TestHandleState_DeploymentDelete(t *testing.T) {
	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	existingRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace, existingRS)

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						K8SNamespace: "test-namespace",
						K8SName:      "test-deployment",
					},
				},
			},
		},
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)

	_, err = client.AppsV1().ReplicaSets("test-namespace").Get(ctx, "test-deployment", metav1.GetOptions{})
	require.Error(t, err, "ReplicaSet should have been deleted")
}

func TestHandleState_SentinelApply(t *testing.T) {
	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)

	var appliedDeployment *appsv1.Deployment
	var appliedService *corev1.Service

	client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var dep appsv1.Deployment
		if err := json.Unmarshal(patchAction.GetPatch(), &dep); err != nil {
			return true, nil, err
		}

		appliedDeployment = &dep
		dep.Namespace = patchAction.GetNamespace()
		return true, &dep, nil
	})

	client.PrependReactor("patch", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var svc corev1.Service
		if err := json.Unmarshal(patchAction.GetPatch(), &svc); err != nil {
			return true, nil, err
		}

		appliedService = &svc
		svc.Namespace = patchAction.GetNamespace()
		return true, &svc, nil
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Apply{
					Apply: &ctrlv1.ApplySentinel{
						WorkspaceId:   "ws_123",
						ProjectId:     "prj_123",
						EnvironmentId: "env_123",
						SentinelId:    "sentinel_123",
						K8SNamespace:  "test-namespace",
						K8SName:       "test-sentinel",
						Image:         "sentinel:1.0",
						Replicas:      2,
						CpuMillicores: 100,
						MemoryMib:     128,
					},
				},
			},
		},
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)

	require.NotNil(t, appliedDeployment, "ApplySentinel should have created a Deployment")
	require.Equal(t, "test-sentinel", appliedDeployment.Name)

	require.NotNil(t, appliedService, "ApplySentinel should have created a Service")
	require.Equal(t, "test-sentinel", appliedService.Name)
}

func TestHandleState_SentinelDelete(t *testing.T) {
	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	existingRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace, existingService, existingRS)

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Delete{
					Delete: &ctrlv1.DeleteSentinel{
						K8SNamespace: "test-namespace",
						K8SName:      "test-sentinel",
					},
				},
			},
		},
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)

	_, err = client.CoreV1().Services("test-namespace").Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.Error(t, err, "Service should have been deleted")

	_, err = client.AppsV1().ReplicaSets("test-namespace").Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.Error(t, err, "ReplicaSet should have been deleted")
}

func TestHandleState_UnknownStateType(t *testing.T) {
	ctx := context.Background()

	client := fake.NewSimpleClientset()

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	state := &ctrlv1.State{
		Kind: nil,
	}

	err := r.HandleState(ctx, state)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown state type")
}

func TestHandleState_NilState(t *testing.T) {
	ctx := context.Background()

	client := fake.NewSimpleClientset()

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.HandleState(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "state is nil")
}
