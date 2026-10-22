package deployment

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestDeleteDeployment_RemovesSecretIdempotently(t *testing.T) {
	for _, tc := range []struct {
		name   string
		legacy bool
	}{
		{name: "deployment"},
		{name: "legacy", legacy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := fullApplyRequest(t)
			client := fake.NewClientset()
			ctrl := New(Config{ClientSet: client, Cluster: &testutil.MockClusterClient{}, Fingerprints: cache.NewNoopCache[string, string]()})
			if tc.legacy {
				_, err := client.AppsV1().ReplicaSets(req.GetK8SNamespace()).Create(t.Context(), legacyReplicaSet(t, 1), metav1.CreateOptions{})
				require.NoError(t, err)
			} else {
				_, err := client.AppsV1().Deployments(req.GetK8SNamespace()).Create(t.Context(), ctrl.buildDeployment(req, true), metav1.CreateOptions{})
				require.NoError(t, err)
			}
			require.NoError(t, ctrl.ensureDeploymentSecret(t.Context(), req.GetK8SNamespace(), req.GetDeploymentId(), map[string]string{"TOKEN": "value"}))
			for range 2 {
				require.NoError(t, ctrl.DeleteDeployment(t.Context(), &ctrlv1.DeleteDeployment{K8SNamespace: req.GetK8SNamespace(), K8SName: req.GetK8SName()}))
				_, err := client.CoreV1().Secrets(req.GetK8SNamespace()).Get(t.Context(), deploymentResourcePrefix(req.GetDeploymentId()), metav1.GetOptions{})
				require.True(t, apierrors.IsNotFound(err))
			}
		})
	}
}

func TestDesiredResync_SecretOnlyCleanupAndIsolation(t *testing.T) {
	for _, outcome := range []string{"not-found", "delete", "apply", "unavailable"} {
		t.Run(outcome, func(t *testing.T) {
			id := uid.New(uid.DeploymentPrefix)
			namespace := uid.DNS1035()
			client := fake.NewClientset()
			ctrl := New(Config{ClientSet: client, Cluster: &testutil.MockClusterClient{
				GetDesiredDeploymentStateFunc: func(_ context.Context, req *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
					require.Equal(t, id, req.GetDeploymentId())
					switch outcome {
					case "not-found":
						return nil, connect.NewError(connect.CodeNotFound, errors.New("removed"))
					case "delete":
						return &ctrlv1.DeploymentState{State: &ctrlv1.DeploymentState_Delete{Delete: &ctrlv1.DeleteDeployment{K8SName: uid.DNS1035(), K8SNamespace: namespace}}}, nil
					case "apply":
						return &ctrlv1.DeploymentState{State: &ctrlv1.DeploymentState_Apply{Apply: &ctrlv1.ApplyDeployment{}}}, nil
					default:
						return nil, connect.NewError(connect.CodeUnavailable, errors.New("offline"))
					}
				},
			}})
			require.NoError(t, ctrl.ensureDeploymentSecret(t.Context(), namespace, id, map[string]string{"TOKEN": "value"}))
			for _, secret := range []*corev1.Secret{
				{ObjectMeta: metav1.ObjectMeta{Name: uid.DNS1035(), Namespace: namespace, Labels: labels.New().ManagedByKrane()}, Type: corev1.SecretTypeDockerConfigJson},
				{ObjectMeta: metav1.ObjectMeta{Name: deploymentResourcePrefix(id), Namespace: uid.DNS1035()}, Type: corev1.SecretTypeOpaque},
				{ObjectMeta: metav1.ObjectMeta{Name: uid.DNS1035(), Namespace: namespace, Labels: labels.New().ManagedByKrane().DeploymentID(id)}, Type: corev1.SecretTypeOpaque},
			} {
				_, err := client.CoreV1().Secrets(secret.Namespace).Create(t.Context(), secret, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			ctrl.forEachWorkload(t.Context(), func(context.Context, workload) { t.Error("secrets must not be reported as workloads") })
			fail := true
			client.PrependReactor("delete", "secrets", func(ktesting.Action) (bool, runtime.Object, error) {
				if fail {
					return true, nil, errors.New("temporary failure")
				}
				return false, nil, nil
			})
			ctrl.resyncDeploymentSecrets(t.Context())
			_, err := client.CoreV1().Secrets(namespace).Get(t.Context(), deploymentResourcePrefix(id), metav1.GetOptions{})
			require.NoError(t, err)
			fail = false
			for range 2 {
				ctrl.resyncDeploymentSecrets(t.Context())
			}
			remaining, err := client.CoreV1().Secrets("").List(t.Context(), metav1.ListOptions{})
			require.NoError(t, err)
			if outcome == "apply" || outcome == "unavailable" {
				require.Len(t, remaining.Items, 4)
			} else {
				require.Len(t, remaining.Items, 3)
				_, err := client.CoreV1().Secrets(namespace).Get(t.Context(), deploymentResourcePrefix(id), metav1.GetOptions{})
				require.True(t, apierrors.IsNotFound(err))
			}
		})
	}
}
