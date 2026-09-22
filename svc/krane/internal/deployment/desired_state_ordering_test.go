package deployment

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDesiredStateRevision_StaleApplyCannotRecreateDeletedDeployment(t *testing.T) {
	client := fake.NewSimpleClientset()
	cluster := &testutil.MockClusterClient{}
	controller := New(Config{
		ClientSet:    client,
		Cluster:      cluster,
		Fingerprints: cache.NewNoopCache[string, string](),
	})
	require.NoError(t, controller.DeleteDeployment(t.Context(), &ctrlv1.DeleteDeployment{
		DeploymentId: "dep_1",
		K8SNamespace: "workspace",
		K8SName:      "deployment",
		Revision:     1,
	}))

	err := controller.ApplyDeployment(t.Context(), &ctrlv1.ApplyDeployment{
		DeploymentId: "dep_1",
		Revision:     0,
	})

	require.NoError(t, err, "stale apply must be rejected before it reaches Kubernetes validation")
	require.Len(t, cluster.ReportDeploymentStatusCalls, 1)
}

func TestDesiredStateRevision_FailedApplyCanRetryOlderDelete(t *testing.T) {
	client := fake.NewSimpleClientset()
	cluster := &testutil.MockClusterClient{}
	controller := New(Config{
		ClientSet:    client,
		Cluster:      cluster,
		Fingerprints: cache.NewNoopCache[string, string](),
	})
	require.Error(t, controller.ApplyDeployment(t.Context(), &ctrlv1.ApplyDeployment{
		DeploymentId: "dep_1",
		Revision:     2,
	}))

	require.NoError(t, controller.DeleteDeployment(t.Context(), &ctrlv1.DeleteDeployment{
		DeploymentId: "dep_1",
		K8SNamespace: "workspace",
		K8SName:      "deployment",
		Revision:     1,
	}))
	require.Len(t, cluster.ReportDeploymentStatusCalls, 1)
}

func TestDesiredStateRevision_FailedDeleteCanRetryOlderRevision(t *testing.T) {
	req := fullApplyRequest(t)
	replicaSet := testController().buildReplicaSet(req, false)
	client := fake.NewSimpleClientset(replicaSet)
	cluster := &testutil.MockClusterClient{}
	controller := New(Config{
		ClientSet:    client,
		Cluster:      cluster,
		Region:       "local",
		Platform:     "test",
		Fingerprints: cache.NewNoopCache[string, string](),
	})

	reportErr := errors.New("report failed")
	cluster.ReportDeploymentStatusFunc = func(context.Context, *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
		return nil, reportErr
	}
	deleteRequest := &ctrlv1.DeleteDeployment{
		DeploymentId: testDeploymentID,
		K8SNamespace: testNamespace,
		K8SName:      testK8sName,
		Revision:     3,
	}
	require.ErrorIs(t, controller.DeleteDeployment(t.Context(), deleteRequest), reportErr)

	_, err := client.AppsV1().ReplicaSets(testNamespace).Create(t.Context(), replicaSet, metav1.CreateOptions{})
	require.NoError(t, err)
	cluster.ReportDeploymentStatusFunc = nil
	deleteRequest.Revision = 2
	require.NoError(t, controller.DeleteDeployment(t.Context(), deleteRequest))
	_, err = client.AppsV1().ReplicaSets(testNamespace).Get(t.Context(), testK8sName, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
}
