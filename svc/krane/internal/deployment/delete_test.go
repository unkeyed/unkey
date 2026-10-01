package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	deleteTestDeploymentID = "dep_test"
	deleteTestNamespace    = "ns_test"
	deleteTestRSName       = "rs_test"
)

func TestPermanentDeleteWaitsForPods(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "pod_pending",
		Namespace: deleteTestNamespace,
		Labels:    map[string]string{labels.LabelKeyDeploymentID: deleteTestDeploymentID},
	}}
	client := fake.NewSimpleClientset(pod)
	cluster := &testutil.MockClusterClient{}
	controller := newDeleteTestController(client, cluster)

	err := controller.DeleteDeployment(context.Background(), permanentDelete())
	require.NoError(t, err)
	require.Empty(t, cluster.ReportDeploymentStatusCalls)
	_, err = client.CoreV1().Pods(deleteTestNamespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
}

func TestPermanentDeleteWaitsForTerminatingPod(t *testing.T) {
	pod := deploymentPod("pod_terminating")
	pod.Finalizers = []string{"test.unkey.com/hold"}
	client := fake.NewSimpleClientset(pod)
	client.PrependReactor("delete", "pods", preservePodDeletion(client))
	cluster := &testutil.MockClusterClient{}
	controller := newDeleteTestController(client, cluster)

	require.NoError(t, controller.DeleteDeployment(context.Background(), permanentDelete()))
	require.Empty(t, cluster.ReportDeploymentStatusCalls)
	terminating, err := client.CoreV1().Pods(deleteTestNamespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, terminating.DeletionTimestamp)
	require.Equal(t, pod.Finalizers, terminating.Finalizers)
}

func TestPermanentDeleteWaitsForForegroundReplicaSet(t *testing.T) {
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name:       deleteTestRSName,
		Namespace:  deleteTestNamespace,
		Finalizers: []string{"test.unkey.com/hold"},
	}}
	client := fake.NewSimpleClientset(rs)
	client.PrependReactor("delete", "replicasets", preserveReplicaSetDeletion(client))
	cluster := &testutil.MockClusterClient{}
	controller := newDeleteTestController(client, cluster)

	require.NoError(t, controller.DeleteDeployment(context.Background(), permanentDelete()))
	require.Empty(t, cluster.ReportDeploymentStatusCalls)
	terminating, err := client.AppsV1().ReplicaSets(deleteTestNamespace).Get(context.Background(), rs.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, terminating.DeletionTimestamp)
	require.Equal(t, rs.Finalizers, terminating.Finalizers)
}

func TestPermanentDeleteAPIErrorWithholdsConfirmation(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("temporary pod list failure")
	})
	cluster := &testutil.MockClusterClient{}
	controller := newDeleteTestController(client, cluster)

	err := controller.DeleteDeployment(context.Background(), permanentDelete())
	require.ErrorContains(t, err, "temporary pod list failure")
	require.Empty(t, cluster.ReportDeploymentStatusCalls)
}

func TestPermanentDeleteRemovesOrphanPodBeforeConfirming(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "pod_orphan",
		Namespace: deleteTestNamespace,
		Labels:    map[string]string{labels.LabelKeyDeploymentID: deleteTestDeploymentID},
	}}
	client := fake.NewSimpleClientset(pod)
	cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("deployment missing"))
	}}
	controller := newDeleteTestController(client, cluster)

	require.NoError(t, controller.ReconcileDeployment(context.Background(), permanentDelete()))
	require.Empty(t, cluster.ReportDeploymentStatusCalls)
	require.NoError(t, controller.ReconcileDeployment(context.Background(), permanentDelete()))
	require.Len(t, cluster.ReportDeploymentStatusCalls, 1)
	require.True(t, cluster.ReportDeploymentStatusCalls[0].GetDelete().GetRemovalConfirmed())
	require.Equal(t, deleteTestDeploymentID, cluster.ReportDeploymentStatusCalls[0].GetDelete().GetDeploymentId())
}

func TestPermanentDeleteRetriesLostConfirmation(t *testing.T) {
	reports := 0
	cluster := &testutil.MockClusterClient{ReportDeploymentStatusFunc: func(context.Context, *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
		reports++
		if reports == 1 {
			return nil, errors.New("report lost")
		}
		return &ctrlv1.ReportDeploymentStatusResponse{}, nil
	}}
	controller := newDeleteTestController(fake.NewSimpleClientset(), cluster)

	require.Error(t, controller.DeleteDeployment(context.Background(), permanentDelete()))
	require.NoError(t, controller.DeleteDeployment(context.Background(), permanentDelete()))
	require.Equal(t, 2, reports)
}

func TestReconcileStaleApplyUsesCurrentPermanentDelete(t *testing.T) {
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: deleteTestRSName, Namespace: deleteTestNamespace}}
	client := fake.NewSimpleClientset(rs)
	cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
		return &ctrlv1.DeploymentState{State: &ctrlv1.DeploymentState_Delete{Delete: permanentDelete()}}, nil
	}}
	controller := newDeleteTestController(client, cluster)

	err := controller.ReconcileDeployment(context.Background(), &ctrlv1.DeleteDeployment{
		DeploymentId: deleteTestDeploymentID,
		K8SNamespace: deleteTestNamespace,
		K8SName:      deleteTestRSName,
	})
	require.NoError(t, err)
	_, err = client.AppsV1().ReplicaSets(deleteTestNamespace).Get(context.Background(), deleteTestRSName, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
}

func TestReconcileControlPlaneErrorDoesNotDelete(t *testing.T) {
	for _, code := range []connect.Code{connect.CodeUnavailable, connect.CodeFailedPrecondition, connect.CodeUnauthenticated} {
		t.Run(code.String(), func(t *testing.T) {
			rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: deleteTestRSName, Namespace: deleteTestNamespace}}
			client := fake.NewSimpleClientset(rs, deploymentPod("live-pod"))
			cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
				return nil, connect.NewError(code, errors.New("cluster unavailable or unregistered"))
			}}
			controller := newDeleteTestController(client, cluster)
			err := controller.ReconcileDeployment(context.Background(), permanentDelete())
			require.Error(t, err)
			require.Empty(t, client.Actions(), "cluster errors must not change Kubernetes resources")
			require.Empty(t, cluster.ReportDeploymentStatusCalls)
		})
	}
}

func TestReconcileSerializesStaleApplyDeleteInterleaving(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	client := fake.NewSimpleClientset()
	applyStarted := make(chan struct{})
	releaseApply := make(chan struct{})
	client.PrependReactor("patch", "replicasets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		close(applyStarted)
		select {
		case <-releaseApply:
		case <-ctx.Done():
			return true, nil, ctx.Err()
		}
		rs := &appsv1.ReplicaSet{}
		if err := json.Unmarshal(action.(k8stesting.PatchAction).GetPatch(), rs); err != nil {
			return true, nil, err
		}
		return true, rs, client.Tracker().Create(appsv1.SchemeGroupVersion.WithResource("replicasets"), rs, deleteTestNamespace)
	})
	applyError := errors.New("HPA unavailable after ReplicaSet creation")
	client.PrependReactor("patch", "horizontalpodautoscalers", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, applyError
	})

	apply := fullApplyRequest(t)
	apply.DeploymentId = deleteTestDeploymentID
	apply.K8SName = deleteTestRSName
	apply.K8SNamespace = deleteTestNamespace
	apply.EncryptedEnvironmentVariables = nil
	var desiredReads atomic.Int32
	cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
		if desiredReads.Add(1) == 1 {
			return &ctrlv1.DeploymentState{State: &ctrlv1.DeploymentState_Apply{Apply: apply}}, nil
		}
		return &ctrlv1.DeploymentState{State: &ctrlv1.DeploymentState_Delete{Delete: permanentDelete()}}, nil
	}}
	controller := newDeleteTestController(clientWithCoreREST{
		Interface: client,
		core: coreWithREST{
			CoreV1Interface: client.CoreV1(),
			rest: &restfake.RESTClient{
				GroupVersion: corev1.SchemeGroupVersion, NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
				Client: restfake.CreateHTTPClient(func(*http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
						Body: io.NopCloser(strings.NewReader(`{"apiVersion":"v1","kind":"ServiceAccount","metadata":{"name":"default"}}`))}, nil
				}),
			},
		},
	}, cluster)

	firstDone := make(chan error, 1)
	go func() { firstDone <- controller.ReconcileDeployment(ctx, permanentDelete()) }()
	select {
	case <-applyStarted:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	secondDone := make(chan error, 1)
	go func() { secondDone <- controller.ReconcileDeployment(ctx, permanentDelete()) }()

	time.Sleep(20 * time.Millisecond)
	require.Equal(t, int32(1), desiredReads.Load())
	close(releaseApply)
	require.ErrorIs(t, <-firstDone, applyError)
	require.NoError(t, <-secondDone)
	require.Equal(t, int32(2), desiredReads.Load())
	require.Len(t, cluster.ReportDeploymentStatusCalls, 1)
	require.True(t, cluster.ReportDeploymentStatusCalls[0].GetDelete().GetRemovalConfirmed())
	_, err := client.AppsV1().ReplicaSets(deleteTestNamespace).Get(ctx, deleteTestRSName, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
}

func TestOrphanPodInventoryReconcilesPermanentDelete(t *testing.T) {
	pod := deploymentPod("pod_orphan_inventory")
	pod.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "apps/v1",
		Kind:       "ReplicaSet",
		Name:       deleteTestRSName,
		Controller: new(true),
	}}
	client := fake.NewSimpleClientset(pod)
	cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("deployment missing"))
	}}
	controller := newDeleteTestController(client, cluster)

	controller.forEachDeploymentPod(context.Background(), controller.reconcileOrphanPod)
	require.Empty(t, cluster.ReportDeploymentStatusCalls)
	_, err := client.CoreV1().Pods(deleteTestNamespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))

	controller.handlePodEvent(context.Background(), pod, watch.Deleted)
	require.Len(t, cluster.ReportDeploymentStatusCalls, 1)
	require.True(t, cluster.ReportDeploymentStatusCalls[0].GetDelete().GetRemovalConfirmed())
}

func permanentDelete() *ctrlv1.DeleteDeployment {
	return &ctrlv1.DeleteDeployment{
		DeploymentId: deleteTestDeploymentID,
		K8SNamespace: deleteTestNamespace,
		K8SName:      deleteTestRSName,
		Permanent:    true,
	}
}

func TestOwnerlessPodInventoryRequiresAuthoritativeAbsence(t *testing.T) {
	for _, code := range []connect.Code{connect.CodeNotFound, connect.CodeUnavailable} {
		t.Run(code.String(), func(t *testing.T) {
			pod := deploymentPod("ownerless")
			client := fake.NewSimpleClientset(pod)
			cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
				return nil, connect.NewError(code, errors.New("lookup failed"))
			}}
			controller := newDeleteTestController(client, cluster)
			controller.forEachDeploymentPod(t.Context(), controller.reconcileOrphanPod)
			_, err := client.CoreV1().Pods(deleteTestNamespace).Get(t.Context(), pod.Name, metav1.GetOptions{})
			if code == connect.CodeUnavailable {
				require.NoError(t, err)
				require.Empty(t, cluster.ReportDeploymentStatusCalls)
				return
			}
			require.True(t, apierrors.IsNotFound(err))
			require.Empty(t, cluster.ReportDeploymentStatusCalls)
			controller.handlePodEvent(t.Context(), pod, watch.Deleted)
			require.Len(t, cluster.ReportDeploymentStatusCalls, 1)
			require.True(t, cluster.ReportDeploymentStatusCalls[0].GetDelete().GetRemovalConfirmed())
		})
	}
}

func newDeleteTestController(client kubernetes.Interface, cluster *testutil.MockClusterClient) *Controller {
	return New(Config{
		ClientSet:     client,
		DynamicClient: fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{}),
		Cluster:       cluster,
		Fingerprints:  cache.NewNoopCache[string, string](),
	})
}

func deploymentPod(name string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: deleteTestNamespace,
		Labels: labels.New().
			DeploymentID(deleteTestDeploymentID).
			ManagedByKrane().
			ComponentDeployment(),
	}}
}

func preservePodDeletion(client *fake.Clientset) k8stesting.ReactionFunc {
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteAction := action.(k8stesting.DeleteAction)
		pod, err := client.Tracker().Get(corev1.SchemeGroupVersion.WithResource("pods"), deleteTestNamespace, deleteAction.GetName())
		if err != nil {
			return true, nil, err
		}
		terminating := pod.(*corev1.Pod).DeepCopy()
		now := metav1.Now()
		terminating.DeletionTimestamp = &now
		return true, nil, client.Tracker().Update(corev1.SchemeGroupVersion.WithResource("pods"), terminating, deleteTestNamespace)
	}
}

func preserveReplicaSetDeletion(client *fake.Clientset) k8stesting.ReactionFunc {
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteAction := action.(k8stesting.DeleteAction)
		rs, err := client.Tracker().Get(appsv1.SchemeGroupVersion.WithResource("replicasets"), deleteTestNamespace, deleteAction.GetName())
		if err != nil {
			return true, nil, err
		}
		terminating := rs.(*appsv1.ReplicaSet).DeepCopy()
		now := metav1.Now()
		terminating.DeletionTimestamp = &now
		return true, nil, client.Tracker().Update(appsv1.SchemeGroupVersion.WithResource("replicasets"), terminating, deleteTestNamespace)
	}
}

type clientWithCoreREST struct {
	kubernetes.Interface
	core coreclient.CoreV1Interface
}

func (c clientWithCoreREST) CoreV1() coreclient.CoreV1Interface { return c.core }

type coreWithREST struct {
	coreclient.CoreV1Interface
	rest rest.Interface
}

func (c coreWithREST) RESTClient() rest.Interface { return c.rest }
