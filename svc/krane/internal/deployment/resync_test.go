package deployment

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestResyncActualStateReportsCompleteRegionalInventory(t *testing.T) {
	client := fake.NewSimpleClientset(
		newManagedReplicaSet("deployment-z"),
		newManagedReplicaSet("deployment-a"),
		&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "unmanaged"}},
	)
	cluster := &testutil.MockClusterClient{}
	controller := New(Config{
		ClientSet:    client,
		Cluster:      cluster,
		CellID:       "cell_1",
		Platform:     "aws",
		Region:       "us-east-1",
		Fingerprints: cache.NewNoopCache[string, string](),
	})

	controller.resyncActualState(context.Background())

	calls := cluster.DeploymentStatusCalls()
	require.Len(t, calls, 3)
	require.Equal(t, []string{"deployment-a", "deployment-z"}, calls[0].GetInventory().GetDeploymentIds())
	require.Equal(t, &ctrlv1.ClusterKey{CellId: "cell_1", Platform: "aws", Region: "us-east-1"}, calls[0].GetCluster())
}

func TestResyncActualStateReportsEmptyRegionalInventory(t *testing.T) {
	cluster := &testutil.MockClusterClient{}
	controller := New(Config{
		ClientSet:    fake.NewSimpleClientset(),
		Cluster:      cluster,
		CellID:       "cell_1",
		Platform:     "aws",
		Region:       "us-east-1",
		Fingerprints: cache.NewNoopCache[string, string](),
	})

	controller.resyncActualState(context.Background())

	calls := cluster.DeploymentStatusCalls()
	require.Len(t, calls, 1)
	require.NotNil(t, calls[0].GetInventory())
	require.Empty(t, calls[0].GetInventory().GetDeploymentIds())
}

func TestResyncActualStateSkipsInventoryForInvalidDeploymentID(t *testing.T) {
	for _, test := range []struct {
		name        string
		deleteLabel bool
	}{
		{name: "missing", deleteLabel: true},
		{name: "empty"},
	} {
		t.Run(test.name, func(t *testing.T) {
			replicaSet := newManagedReplicaSet("deployment-a")
			if test.deleteLabel {
				delete(replicaSet.Labels, labels.LabelKeyDeploymentID)
			} else {
				replicaSet.Labels[labels.LabelKeyDeploymentID] = ""
			}
			cluster := &testutil.MockClusterClient{}
			controller := New(Config{
				ClientSet:    fake.NewSimpleClientset(replicaSet),
				Cluster:      cluster,
				CellID:       "cell_1",
				Platform:     "aws",
				Region:       "us-east-1",
				Fingerprints: cache.NewNoopCache[string, string](),
			})

			controller.resyncActualState(context.Background())

			calls := cluster.DeploymentStatusCalls()
			require.Len(t, calls, 1)
			require.Nil(t, calls[0].GetInventory())
			require.Equal(t, "deployment-a", calls[0].GetUpdate().GetK8SName())
		})
	}
}

func TestResyncActualStateSkipsInventoryAfterPartialList(t *testing.T) {
	client := fake.NewSimpleClientset()
	listCalls := 0
	client.Fake.PrependReactor("list", "replicasets", func(ktesting.Action) (bool, runtime.Object, error) {
		listCalls++
		if listCalls == 1 {
			return true, &appsv1.ReplicaSetList{
				ListMeta: metav1.ListMeta{Continue: "next-page"},
				Items:    []appsv1.ReplicaSet{*newManagedReplicaSet("deployment-a")},
			}, nil
		}
		return true, nil, errors.New("list failed")
	})
	cluster := &testutil.MockClusterClient{}
	controller := New(Config{
		ClientSet:    client,
		Cluster:      cluster,
		CellID:       "cell_1",
		Platform:     "aws",
		Region:       "us-east-1",
		Fingerprints: cache.NewNoopCache[string, string](),
	})

	controller.resyncActualState(context.Background())

	calls := cluster.DeploymentStatusCalls()
	require.Len(t, calls, 1)
	require.Nil(t, calls[0].GetInventory())
	require.Equal(t, "deployment-a", calls[0].GetUpdate().GetK8SName())
}

func newManagedReplicaSet(name string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID(name),
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"replicaset": name}},
		},
	}
}
