package sentinelcontroller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestShouldCreateNewSentinel(t *testing.T) {
	h := NewTestHarness(t)

	c := h.NewController()

	name := uid.DNS1035()

	// Before reconciling the sentinel, ensure it does not exist
	found := &apiv1.Sentinel{}
	err := h.k8sClient.Get(h.ctx, types.NamespacedName{Namespace: h.namespace, Name: name}, found)
	require.Error(t, err)
	require.True(t, errors.IsNotFound(err), "sentinel should not exist, but it does: %w", err)

	sentinel := &apiv1.Sentinel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: h.namespace,
		},
		Spec: apiv1.SentinelSpec{
			WorkspaceID:   uid.New(uid.TestPrefix),
			ProjectID:     uid.New(uid.TestPrefix),
			EnvironmentID: uid.New(uid.TestPrefix),
			SentinelID:    uid.New(uid.TestPrefix),
			Image:         "nginx:latest",
			Replicas:      3,
			CpuMillicores: 1024,
			MemoryMib:     1024,
		},
	}

	err = h.k8sClient.Create(h.ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(c.reconciler, h.namespace, name)

	// Now it should exist
	err = h.k8sClient.Get(h.ctx, types.NamespacedName{Namespace: h.namespace, Name: name}, found)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, h.k8sClient.Delete(h.ctx, found))

	})

	deployments := &appsv1.DeploymentList{}
	err = h.k8sClient.List(h.ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment := deployments.Items[0]
	require.True(t, strings.HasPrefix(deployment.Name, sentinel.Name))

	services := &corev1.ServiceList{}
	err = h.k8sClient.List(h.ctx, services, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, services.Items, 1)
	service := services.Items[0]
	require.True(t, strings.HasPrefix(service.Name, sentinel.Name))

}

func TestShouldReconcileImageChanges(t *testing.T) {
	h := NewTestHarness(t)

	c := h.NewController()

	name := uid.DNS1035()

	sentinel := &apiv1.Sentinel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: h.namespace,
		},
		Spec: apiv1.SentinelSpec{
			WorkspaceID:   uid.New(uid.TestPrefix),
			ProjectID:     uid.New(uid.TestPrefix),
			EnvironmentID: uid.New(uid.TestPrefix),
			SentinelID:    uid.New(uid.TestPrefix),
			Image:         "nginx:latest",
			Replicas:      3,
			CpuMillicores: 1024,
			MemoryMib:     1024,
		},
	}

	err := h.k8sClient.Create(h.ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(c.reconciler, h.namespace, name)

	deployments := &appsv1.DeploymentList{}
	err = h.k8sClient.List(h.ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment := deployments.Items[0]
	for _, container := range deployment.Spec.Template.Spec.Containers {
		require.Equal(t, sentinel.Spec.Image, container.Image)
	}

	err = h.k8sClient.Get(h.ctx, types.NamespacedName{Namespace: h.namespace, Name: name}, sentinel)
	require.NoError(t, err)
	sentinel.Spec.Image = "nginx:alpine"
	err = h.k8sClient.Update(h.ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(c.reconciler, h.namespace, name)

	err = h.k8sClient.List(h.ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment = deployments.Items[0]
	for _, container := range deployment.Spec.Template.Spec.Containers {
		require.Equal(t, sentinel.Spec.Image, container.Image)
	}

}

func TestShouldReconcileReplicaChanges(t *testing.T) {
	h := NewTestHarness(t)

	c := h.NewController()
	name := uid.DNS1035()

	sentinel := &apiv1.Sentinel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: h.namespace,
		},
		Spec: apiv1.SentinelSpec{
			WorkspaceID:   uid.New(uid.TestPrefix),
			ProjectID:     uid.New(uid.TestPrefix),
			EnvironmentID: uid.New(uid.TestPrefix),
			SentinelID:    uid.New(uid.TestPrefix),
		},
	}

	err := h.k8sClient.Create(h.ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(c.reconciler, h.namespace, name)

	deployments := &appsv1.DeploymentList{}
	err = h.k8sClient.List(h.ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment := deployments.Items[0]
	require.Equal(t, sentinel.Spec.Replicas, *deployment.Spec.Replicas)

	err = h.k8sClient.Get(h.ctx, types.NamespacedName{Namespace: h.namespace, Name: name}, sentinel)
	require.NoError(t, err)
	sentinel.Spec.Replicas = 5
	err = h.k8sClient.Update(h.ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(c.reconciler, h.namespace, name)

	err = h.k8sClient.List(h.ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment = deployments.Items[0]
	require.Equal(t, sentinel.Spec.Replicas, *deployment.Spec.Replicas)

}
