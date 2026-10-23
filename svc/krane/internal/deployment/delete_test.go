package deployment

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestDesiredResync_SecretOnlyCleanupAndIsolation(t *testing.T) {
	for _, outcome := range []string{"not-found", "delete", "apply", "unavailable", "owner-added"} {
		t.Run(outcome, func(t *testing.T) {
			id := uid.New(uid.DeploymentPrefix)
			namespace := uid.DNS1035()
			client := fake.NewClientset()
			owner := metav1.OwnerReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       uid.DNS1035(),
				UID:        types.UID(uid.New(uid.DeploymentPrefix)),
				Controller: new(true),
			}
			ctrl := New(Config{ClientSet: client, Cluster: &testutil.MockClusterClient{
				GetDesiredDeploymentStateFunc: func(_ context.Context, req *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
					require.Equal(t, id, req.GetDeploymentId())
					if outcome == "owner-added" {
						secret, err := client.CoreV1().Secrets(namespace).Get(t.Context(), deploymentResourcePrefix(id), metav1.GetOptions{})
						require.NoError(t, err)
						secret.OwnerReferences = []metav1.OwnerReference{owner}
						_, err = client.CoreV1().Secrets(namespace).Update(t.Context(), secret, metav1.UpdateOptions{})
						require.NoError(t, err)
					}
					switch outcome {
					case "not-found":
						return nil, connect.NewError(connect.CodeNotFound, errors.New("removed"))
					case "delete", "owner-added":
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
				{ObjectMeta: metav1.ObjectMeta{Name: deploymentResourcePrefix(id), Namespace: uid.DNS1035(), Labels: labels.New().ManagedByKrane().DeploymentID(id), OwnerReferences: []metav1.OwnerReference{owner}}, Type: corev1.SecretTypeOpaque},
			} {
				_, err := client.CoreV1().Secrets(secret.Namespace).Create(t.Context(), secret, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			ctrl.forEachWorkload(t.Context(), func(context.Context, workload) { t.Error("secrets must not be reported as workloads") })
			fail := outcome != "owner-added"
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
			if outcome == "apply" || outcome == "unavailable" || outcome == "owner-added" {
				require.Len(t, remaining.Items, 5)
			} else {
				require.Len(t, remaining.Items, 4)
				_, err := client.CoreV1().Secrets(namespace).Get(t.Context(), deploymentResourcePrefix(id), metav1.GetOptions{})
				require.True(t, apierrors.IsNotFound(err))
			}
		})
	}
}
