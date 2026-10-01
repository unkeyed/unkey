package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestDeploymentPodInventoryFollowsPagination(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Query().Get("limit") + ":" + r.URL.Query().Get("continue")
		page := &corev1.PodList{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "PodList"}}
		if r.URL.Query().Get("continue") == "" {
			page.Continue = "next-page"
			page.Items = []corev1.Pod{*deploymentPod("first")}
		} else {
			page.Items = []corev1.Pod{*deploymentPod("last")}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(page); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)
	controller := newDeleteTestController(client, &testutil.MockClusterClient{})
	var names []string
	controller.forEachDeploymentPod(t.Context(), func(_ context.Context, pod *corev1.Pod) { names = append(names, pod.Name) })
	require.Equal(t, []string{"first", "last"}, names)
	require.Equal(t, "500:", <-requests)
	require.Equal(t, "500:next-page", <-requests)
}

func TestDesiredInventoryReusesReplicaSetListForPodOwners(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pod := deploymentPod("owned-pod")
		pod.OwnerReferences = []metav1.OwnerReference{{Kind: "ReplicaSet", Name: deleteTestRSName, Controller: new(true)}}
		rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: deleteTestRSName, Namespace: pod.Namespace, Labels: pod.Labels}}
		client := fake.NewSimpleClientset(rs, pod)
		cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("offline"))
		}}
		controller := newDeleteTestController(client, cluster)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan struct{})
		go func() {
			controller.runDesiredStateResyncLoop(ctx)
			close(done)
		}()
		synctest.Wait()
		cancel()
		<-done
		for _, action := range client.Actions() {
			require.False(t, action.Matches("get", "replicasets"), "owned pods must reuse the inventory, not GET each owner")
		}
	})
}
