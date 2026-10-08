package deployment

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func legacyReplicaSet(t *testing.T, replicas int32) *appsv1.ReplicaSet {
	t.Helper()
	desired := testController().buildDeployment(fullApplyRequest(t), false)
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: desired.Namespace,
			Name:      desired.Name,
			Labels:    desired.Labels,
			UID:       types.UID("legacy-rs-uid"),
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: new(replicas),
			Selector: desired.Spec.Selector,
			Template: desired.Spec.Template,
		},
	}
}

func getDeployment(t *testing.T, ctrl *Controller) *appsv1.Deployment {
	t.Helper()
	d, err := ctrl.clientSet.AppsV1().Deployments(testNamespace).Get(t.Context(), testK8sName, metav1.GetOptions{})
	require.NoError(t, err)
	return d
}

func TestApplyDeploymentObject_NewWorkloadStartsUnpaused(t *testing.T) {
	ctrl := testController()
	ctrl.clientSet = fake.NewClientset()

	_, err := ctrl.applyDeploymentObject(t.Context(), ctrl.buildDeployment(fullApplyRequest(t), false))
	require.NoError(t, err)

	d := getDeployment(t, ctrl)
	require.False(t, d.Spec.Paused, "a new workload must start its pods at once")
	require.Nil(t, d.Spec.Replicas, "the HPA owns the replica count")
}

func TestApplyDeploymentObject_AdoptsLegacyReplicaSet(t *testing.T) {
	ctrl := testController()
	ctrl.clientSet = fake.NewClientset(legacyReplicaSet(t, 4))

	_, err := ctrl.applyDeploymentObject(t.Context(), ctrl.buildDeployment(fullApplyRequest(t), false))
	require.NoError(t, err)

	d := getDeployment(t, ctrl)
	require.True(t, d.Spec.Paused, "adoption must not roll the legacy pods")
	require.Equal(t, new(int32(4)), d.Spec.Replicas, "adoption must keep the legacy replica count")

	req := fullApplyRequest(t)
	req.Image = "registry.test/sentinel-image:v2"
	_, err = ctrl.applyDeploymentObject(t.Context(), ctrl.buildDeployment(req, false))
	require.NoError(t, err)

	d = getDeployment(t, ctrl)
	require.True(t, d.Spec.Paused, "a krane apply must not unpause the deployment")
	require.Equal(t, new(int32(4)), d.Spec.Replicas, "a krane apply must not reset the replica count")
	require.Equal(t, req.Image, mainContainer(t, d).Image)
}

func TestApplyDeploymentObject_KeepsGateDecision(t *testing.T) {
	ctrl := testController()
	ctrl.clientSet = fake.NewClientset(legacyReplicaSet(t, 2))

	_, err := ctrl.applyDeploymentObject(t.Context(), ctrl.buildDeployment(fullApplyRequest(t), false))
	require.NoError(t, err)

	ctrl.setRolloutState(t.Context(), getDeployment(t, ctrl), false, true)

	_, err = ctrl.applyDeploymentObject(t.Context(), ctrl.buildDeployment(fullApplyRequest(t), false))
	require.NoError(t, err)

	d := getDeployment(t, ctrl)
	require.False(t, d.Spec.Paused, "a krane apply must not pause an admitted rollout")
	require.True(t, holdsRolloutSlot(d), "a krane apply must not remove the slot annotation")
}

func TestApplyDeploymentObject_IgnoresControlledReplicaSet(t *testing.T) {
	rs := legacyReplicaSet(t, 3)
	rs.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       "other",
		UID:        types.UID("other-uid"),
		Controller: new(true),
	}}
	ctrl := testController()
	ctrl.clientSet = fake.NewClientset(rs)

	_, err := ctrl.applyDeploymentObject(t.Context(), ctrl.buildDeployment(fullApplyRequest(t), false))
	require.NoError(t, err)

	d := getDeployment(t, ctrl)
	require.False(t, d.Spec.Paused)
	require.Nil(t, d.Spec.Replicas)
}

func TestForEachWorkload_SkipsDeploymentRevisions(t *testing.T) {
	legacy := legacyReplicaSet(t, 1)
	legacy.Name = "legacy"

	d := testController().buildDeployment(fullApplyRequest(t), false)
	d.UID = types.UID("deployment-uid")

	revision := legacyReplicaSet(t, 1)
	revision.Name = testK8sName + "-abc123"
	revision.OwnerReferences = []metav1.OwnerReference{deploymentOwnerRef(d)}

	ctrl := &Controller{clientSet: fake.NewClientset(legacy, d, revision)}

	var mu sync.Mutex
	var got []workload
	ctrl.forEachWorkload(t.Context(), func(_ context.Context, w workload) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, w)
	})

	var gotNames []string
	for _, w := range got {
		gotNames = append(gotNames, w.k8sName)
		require.Equal(t, testDeploymentID, w.deploymentID)
	}
	sort.Strings(gotNames)
	require.Equal(t, []string{"legacy", testK8sName}, gotNames)
}

func TestWorkloadForReplicaSet(t *testing.T) {
	d := testController().buildDeployment(fullApplyRequest(t), false)
	d.UID = types.UID("deployment-uid")

	revision := legacyReplicaSet(t, 1)
	revision.Name = testK8sName + "-abc123"
	revision.OwnerReferences = []metav1.OwnerReference{deploymentOwnerRef(d)}
	revision.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
		labels.LabelKeyDeploymentID: testDeploymentID,
		"pod-template-hash":         "abc123",
	}}

	ctrl := &Controller{clientSet: fake.NewClientset(d)}

	w, err := ctrl.workloadForReplicaSet(t.Context(), revision)
	require.NoError(t, err)
	require.Equal(t, testK8sName, w.k8sName, "status must be reported under the Deployment name")
	require.Equal(t, d.Spec.Selector, w.selector, "status must cover the pods of every revision")

	legacy := legacyReplicaSet(t, 1)
	w, err = ctrl.workloadForReplicaSet(t.Context(), legacy)
	require.NoError(t, err)
	require.Equal(t, legacy.Name, w.k8sName)
}
