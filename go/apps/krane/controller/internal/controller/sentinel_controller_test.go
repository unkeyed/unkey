/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestShouldReconcileSentinel(t *testing.T) {
	h := NewTestHarness(t)

	ctx := context.Background()

	r := &SentinelReconciler{
		Client: h.k8sClient,
		Scheme: h.k8sClient.Scheme(),
	}

	name := uid.DNS1035()

	// Before reconciling the sentinel, ensure it does not exist
	found := &apiv1.Sentinel{}
	err := h.k8sClient.Get(ctx, types.NamespacedName{Namespace: h.namespace, Name: name}, found)
	require.Error(t, err)
	require.True(t, errors.IsNotFound(err))

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

	err = h.k8sClient.Create(ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(r, h.namespace, name)

	// Now it should exist
	err = h.k8sClient.Get(ctx, types.NamespacedName{Namespace: h.namespace, Name: name}, found)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, h.k8sClient.Delete(ctx, found))

	})

	deployments := &appsv1.DeploymentList{}
	err = h.k8sClient.List(ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment := deployments.Items[0]
	require.True(t, strings.HasPrefix(deployment.Name, sentinel.Name))

	services := &corev1.ServiceList{}
	err = h.k8sClient.List(ctx, services, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, services.Items, 1)
	service := services.Items[0]
	require.True(t, strings.HasPrefix(service.Name, sentinel.Name))

}

func TestShouldReconcileImageChanges(t *testing.T) {
	h := NewTestHarness(t)

	ctx := context.Background()

	r := &SentinelReconciler{
		Client: h.k8sClient,
		Scheme: h.k8sClient.Scheme(),
	}

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

	err := h.k8sClient.Create(ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(r, h.namespace, name)

	deployments := &appsv1.DeploymentList{}
	err = h.k8sClient.List(ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment := deployments.Items[0]
	for _, container := range deployment.Spec.Template.Spec.Containers {
		require.Equal(t, sentinel.Spec.Image, container.Image)
	}

	err = h.k8sClient.Get(ctx, types.NamespacedName{Namespace: h.namespace, Name: name}, sentinel)
	require.NoError(t, err)
	sentinel.Spec.Image = "nginx:alpine"
	err = h.k8sClient.Update(ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(r, h.namespace, name)

	err = h.k8sClient.List(ctx, deployments, client.MatchingLabelsSelector{
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

	ctx := context.Background()

	r := &SentinelReconciler{
		Client: h.k8sClient,
		Scheme: h.k8sClient.Scheme(),
	}

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

	err := h.k8sClient.Create(ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(r, h.namespace, name)

	deployments := &appsv1.DeploymentList{}
	err = h.k8sClient.List(ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment := deployments.Items[0]
	require.Equal(t, sentinel.Spec.Replicas, *deployment.Spec.Replicas)

	err = h.k8sClient.Get(ctx, types.NamespacedName{Namespace: h.namespace, Name: name}, sentinel)
	require.NoError(t, err)
	sentinel.Spec.Replicas = 5
	err = h.k8sClient.Update(ctx, sentinel)
	require.NoError(t, err)

	h.FullReconcileOrFail(r, h.namespace, name)

	err = h.k8sClient.List(ctx, deployments, client.MatchingLabelsSelector{
		Selector: k8s.NewLabels().ManagedByKrane().SentinelID(sentinel.Spec.SentinelID).ToSelector(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	deployment = deployments.Items[0]
	require.Equal(t, sentinel.Spec.Replicas, *deployment.Spec.Replicas)

}
