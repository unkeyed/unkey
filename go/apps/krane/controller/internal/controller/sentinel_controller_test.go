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
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/controller/api/v1"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestShouldReconcileSentinel(t *testing.T) {
	h := NewTestHarness(t)

	ctx := context.Background()

	r := &SentinelReconciler{
		Client: h.k8sClient,
		Scheme: h.k8sClient.Scheme(),
	}

	namespace := "default"
	name := uid.NanoLower(8)

	// Before reconciling the sentinel, ensure it does not exist
	found := &apiv1.Sentinel{}
	err := h.k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, found)
	require.Error(t, err)
	require.True(t, errors.IsNotFound(err))

	sentinel := &apiv1.Sentinel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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

	_, err = r.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	})
	require.NoError(t, err)

	// Now it should exist
	err = h.k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, found)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, h.k8sClient.Delete(ctx, found))

	})

}
