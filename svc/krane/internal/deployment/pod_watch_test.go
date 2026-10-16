package deployment

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func podEventFixture(t *testing.T) (*appsv1.Deployment, *appsv1.ReplicaSet, *corev1.Pod) {
	t.Helper()
	d := testController().buildDeployment(fullApplyRequest(t), false)
	d.UID = types.UID("deployment-uid")

	revision := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       testNamespace,
			Name:            testK8sName + "-abc123",
			Labels:          d.Spec.Template.Labels,
			UID:             types.UID("revision-uid"),
			OwnerReferences: []metav1.OwnerReference{deploymentOwnerRef(d)},
		},
		Spec: appsv1.ReplicaSetSpec{Selector: d.Spec.Selector, Template: d.Spec.Template},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testK8sName + "-abc123-x1",
			Labels:    d.Spec.Template.Labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       revision.Name,
				UID:        revision.UID,
				Controller: new(true),
			}},
		},
		Spec: d.Spec.Template.Spec,
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			PodIP:      "10.0.0.7",
			Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}},
		},
	}

	return d, revision, pod
}

func TestRunPodEventHandlerReportsPodChanges(t *testing.T) {
	d, revision, pod := podEventFixture(t)
	reports := make(chan *ctrlv1.ReportDeploymentStatusRequest_Update, 10)
	ctrl := cachedController(t, d, revision, pod)
	ctrl.cluster = &testutil.MockClusterClient{
		ReportDeploymentStatusFunc: func(_ context.Context, req *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
			reports <- req.GetUpdate()
			return &ctrlv1.ReportDeploymentStatusResponse{}, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		ctrl.runPodEventHandler(ctx)
		close(done)
	}()

	nextReport := func() *ctrlv1.ReportDeploymentStatusRequest_Update {
		t.Helper()
		select {
		case update := <-reports:
			return update
		case <-time.After(5 * time.Second):
			t.Fatal("no status report")
			return nil
		}
	}

	update := nextReport()
	require.Equal(t, testK8sName, update.GetK8SName(), "status must be reported under the Deployment name")
	require.Len(t, update.GetInstances(), 1)
	require.Equal(t, pod.Name, update.GetInstances()[0].GetK8SName())

	require.NoError(t, ctrl.clientSet.CoreV1().Pods(testNamespace).Delete(t.Context(), pod.Name, metav1.DeleteOptions{}))
	update = nextReport()
	require.Equal(t, testK8sName, update.GetK8SName())
	require.Empty(t, update.GetInstances(), "a deleted pod must stop being reported")

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pod event handler did not stop after cancellation")
	}
}
