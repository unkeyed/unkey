package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/clientcmd"
)

func TestPolicyOwnershipSurvivesFailedApplyWithAPIServer(t *testing.T) {
	contextName := os.Getenv("UNKEY_CILIUM_TEST_CONTEXT")
	if contextName == "" {
		t.Skip("requires a disposable Kubernetes context with the CiliumNetworkPolicy CRD")
	}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{CurrentContext: contextName}).ClientConfig()
	require.NoError(t, err)
	config.Timeout = 10 * time.Second
	client, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)
	policies, err := dynamic.NewForConfig(config)
	require.NoError(t, err)
	namespace, err := client.CoreV1().Namespaces().Create(t.Context(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "unkey-policy-owner-test-"}}, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 30*time.Second)
		defer cancel()
		require.NoError(t, client.CoreV1().Namespaces().Delete(ctx, namespace.Name, metav1.DeleteOptions{}))
	})
	req := fullApplyRequest(t)
	req.K8SNamespace = namespace.Name
	req.EncryptedEnvironmentVariables = nil
	controller := New(Config{ClientSet: client, DynamicClient: policies})
	require.NoError(t, controller.ensureCiliumNetworkPolicy(t.Context(), req, nil))
	replicaSet, err := client.AppsV1().ReplicaSets(namespace.Name).Create(t.Context(), &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: req.GetK8SName()},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: new(int32(0)), Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "immutable-selector"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"test": "immutable-selector"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "unused", Image: "unused"}}},
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, controller.ensureCiliumNetworkPolicy(t.Context(), req, replicaSet))
	err = controller.ApplyDeployment(t.Context(), req)
	require.True(t, apierrors.IsConflict(err), "selector ownership must reject the ReplicaSet write: %v", err)
	policy, err := policies.Resource(cilium.NetworkPolicyResource).Namespace(namespace.Name).Get(t.Context(), req.GetK8SName()+frontlinePolicySuffix, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, []metav1.OwnerReference{replicaSetOwnerRef(replicaSet)}, policy.GetOwnerReferences())
	require.NoError(t, client.AppsV1().ReplicaSets(namespace.Name).Delete(t.Context(), replicaSet.Name, metav1.DeleteOptions{}))
	require.Eventually(t, func() bool {
		_, err := policies.Resource(cilium.NetworkPolicyResource).Namespace(namespace.Name).Get(t.Context(), policy.GetName(), metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}, 20*time.Second, 100*time.Millisecond)
}

func TestApplyDeploymentPreservesPolicyOwnershipAcrossFailures(t *testing.T) {
	for _, failure := range []string{"hpa", "replicaset rejected", "replicaset response lost"} {
		t.Run(failure, func(t *testing.T) {
			policies := policyTestClient()
			var failReplicaSet atomic.Bool
			failReplicaSet.Store(failure != "hpa")
			replicaSetUID := types.UID(uid.New(uid.TestPrefix))
			var replicaSet *appsv1.ReplicaSet
			var mu sync.Mutex
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				var response any
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/v1/namespaces":
					response = &corev1.Namespace{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"}, ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
				case strings.HasSuffix(r.URL.Path, "/serviceaccounts/default"):
					response = &corev1.ServiceAccount{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"}, ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: testNamespace}}
				case strings.HasSuffix(r.URL.Path, "/replicasets/"+testK8sName):
					switch r.Method {
					case http.MethodGet:
						if replicaSet == nil {
							w.WriteHeader(http.StatusNotFound)
							response = &metav1.Status{Reason: metav1.StatusReasonNotFound, Code: http.StatusNotFound}
						} else {
							response = replicaSet
						}
					case http.MethodPatch:
						_, err := policies.Resource(cilium.NetworkPolicyResource).Namespace(testNamespace).Get(r.Context(), testK8sName+frontlinePolicySuffix, metav1.GetOptions{})
						if err != nil {
							t.Error("ReplicaSet created before ingress policy", err)
						}
						if !failReplicaSet.Load() || failure == "replicaset response lost" {
							replicaSet = new(appsv1.ReplicaSet)
							if err := json.NewDecoder(r.Body).Decode(replicaSet); err != nil {
								t.Error(err)
								w.WriteHeader(http.StatusBadRequest)
								return
							}
							replicaSet.UID = replicaSetUID
						}
						if failReplicaSet.Load() {
							w.WriteHeader(http.StatusGatewayTimeout)
							response = &metav1.Status{Reason: metav1.StatusReasonTimeout, Code: http.StatusGatewayTimeout}
						} else {
							response = replicaSet
						}
					default:
						t.Errorf("unexpected ReplicaSet request: %s", r.Method)
					}
				case strings.Contains(r.URL.Path, "/horizontalpodautoscalers/"):
					w.WriteHeader(http.StatusForbidden)
					response = &metav1.Status{Reason: metav1.StatusReasonForbidden, Code: http.StatusForbidden}
				default:
					t.Errorf("unexpected Kubernetes request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
			require.NoError(t, err)
			req := fullApplyRequest(t)
			req.EncryptedEnvironmentVariables = nil
			cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(_ context.Context, request *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
				if request.GetDeploymentId() != testDeploymentID {
					return nil, fmt.Errorf("wrong deployment ID: %s", request.GetDeploymentId())
				}
				return &ctrlv1.DeploymentState{State: &ctrlv1.DeploymentState_Apply{Apply: req}}, nil
			}}
			cfg := Config{ClientSet: client, DynamicClient: policies, Cluster: cluster}
			controller := New(cfg)
			err = controller.ApplyDeployment(t.Context(), req)
			if failure == "hpa" {
				require.True(t, apierrors.IsForbidden(err), "%v", err)
			} else {
				require.True(t, apierrors.IsTimeout(err), "%v", err)
			}
			policy, err := policies.Resource(cilium.NetworkPolicyResource).Namespace(testNamespace).Get(t.Context(), testK8sName+frontlinePolicySuffix, metav1.GetOptions{})
			require.NoError(t, err, "an ambiguous create must never roll back the ingress policy")
			if failure == "hpa" {
				require.Len(t, policy.GetOwnerReferences(), 1, "ownership must precede the failing HPA write")
				failReplicaSet.Store(true)
				require.Error(t, controller.ApplyDeployment(t.Context(), req))
			} else {
				require.Empty(t, policy.GetOwnerReferences())
				failReplicaSet.Store(false)
				controller = New(cfg)
				controller.reconcileUnownedPolicies(t.Context())
			}
			policy, err = policies.Resource(cilium.NetworkPolicyResource).Namespace(testNamespace).Get(t.Context(), testK8sName+frontlinePolicySuffix, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, []metav1.OwnerReference{{
				APIVersion: "apps/v1", Kind: "ReplicaSet", Name: testK8sName, UID: replicaSetUID,
				Controller: new(true), BlockOwnerDeletion: new(true),
			}}, policy.GetOwnerReferences())
		})
	}
}

func TestPolicyWithoutReplicaSetDoesNotPrunePreviousOwner(t *testing.T) {
	policy := applyPolicy(t, true, "preview", "10.0.0.53")
	owners := []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: testK8sName, UID: types.UID(uid.New(uid.TestPrefix))}}
	policy.SetOwnerReferences(owners)
	client := policyTestClient(policy)
	controller := &Controller{dynamicClient: client}
	require.Error(t, controller.ensureCiliumNetworkPolicy(t.Context(), fullApplyRequest(t), nil))
	stored, err := client.Resource(cilium.NetworkPolicyResource).Namespace(testNamespace).Get(t.Context(), policy.GetName(), metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, owners, stored.GetOwnerReferences())
	for _, action := range client.Actions() {
		require.NotEqual(t, "patch", action.GetVerb())
	}
}

// TestDesiredStateReadSerializesWithDeletion guarantees that a resync which
// read a running desired state before a deletion cannot apply it afterwards.
func TestDesiredStateReadSerializesWithDeletion(t *testing.T) {
	reading, release := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	t.Cleanup(unblock)
	running := fullApplyRequest(t)
	cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
		close(reading)
		<-release
		return &ctrlv1.DeploymentState{State: &ctrlv1.DeploymentState_Apply{Apply: running}}, nil
	}}
	client := fake.NewClientset()
	client.PrependReactor("create", "namespaces", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("stop the stale apply at its first write")
	})
	deleting := make(chan struct{}, 1)
	client.PrependReactor("delete", "replicasets", func(clienttesting.Action) (bool, runtime.Object, error) {
		deleting <- struct{}{}
		return false, nil, nil
	})
	controller := New(Config{
		ClientSet: client, DynamicClient: policyTestClient(), Cluster: cluster,
		Fingerprints: cache.NewNoopCache[string, string](),
	})
	reconciled := make(chan struct{})
	go func() {
		controller.reconcileDesiredState(t.Context(), testNamespace, testK8sName, testDeploymentID)
		close(reconciled)
	}()
	<-reading
	started, deleted := make(chan struct{}), make(chan error, 1)
	go func() {
		close(started)
		deleted <- controller.DeleteDeployment(t.Context(), &ctrlv1.DeleteDeployment{K8SNamespace: testNamespace, K8SName: testK8sName})
	}()
	<-started
	select {
	case <-deleting:
		t.Error("deletion raced an in-flight desired-state read")
	case <-time.After(50 * time.Millisecond):
	}
	unblock()
	<-reconciled
	require.NoError(t, <-deleted)
	var verbs []string
	for _, action := range client.Actions() {
		verbs = append(verbs, action.GetVerb()+" "+action.GetResource().Resource)
	}
	require.Equal(t, []string{"create namespaces", "delete replicasets"}, verbs, "the stale apply must finish before the deletion starts")
}

func TestUnownedPolicyResyncRequiresAuthoritativeDeletion(t *testing.T) {
	for _, code := range []connect.Code{connect.CodeNotFound, connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeInternal} {
		t.Run(code.String(), func(t *testing.T) {
			policy := applyPolicy(t, true, "preview", "10.0.0.53")
			policy.SetUID("policy-uid")
			policy.SetResourceVersion("17")
			policies := policyTestClient(policy)
			cluster := &testutil.MockClusterClient{GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
				return nil, connect.NewError(code, errors.New("control plane response"))
			}}
			controller := New(Config{
				ClientSet: fake.NewClientset(), DynamicClient: policies, Cluster: cluster,
				Fingerprints: cache.NewNoopCache[string, string](),
			})
			controller.reconcileUnownedPolicies(t.Context())
			_, err := policies.Resource(cilium.NetworkPolicyResource).Namespace(testNamespace).Get(t.Context(), policy.GetName(), metav1.GetOptions{})
			if code == connect.CodeNotFound {
				require.True(t, apierrors.IsNotFound(err))
				for _, action := range policies.Actions() {
					if action.GetVerb() == "delete" {
						options := action.(clienttesting.DeleteAction).GetDeleteOptions()
						require.Equal(t, &metav1.Preconditions{UID: new(types.UID("policy-uid")), ResourceVersion: new("17")}, options.Preconditions)
					}
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUnownedPolicyCleanupWaitsForWorkloads(t *testing.T) {
	for _, state := range []string{"replicaset", "terminating pod", "pod list unavailable", "replicaset read unavailable", "empty"} {
		t.Run(state, func(t *testing.T) {
			policy := applyPolicy(t, true, "preview", "10.0.0.53")
			policies := policyTestClient(policy)
			client := fake.NewClientset()
			switch state {
			case "replicaset":
				_, err := client.AppsV1().ReplicaSets(testNamespace).Create(t.Context(), &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: testK8sName}}, metav1.CreateOptions{})
				require.NoError(t, err)
			case "terminating pod":
				_, err := client.CoreV1().Pods(testNamespace).Create(t.Context(), &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Name: "terminating", Labels: labels.New().DeploymentID(testDeploymentID), DeletionTimestamp: new(metav1.Now()),
				}}, metav1.CreateOptions{})
				require.NoError(t, err)
			case "pod list unavailable":
				client.PrependReactor("list", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("pod list unavailable")
				})
			case "replicaset read unavailable":
				client.PrependReactor("get", "replicasets", func(clienttesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("replicaset read unavailable")
				})
			}
			controller := &Controller{clientSet: client, dynamicClient: policies}
			err := controller.deleteUnownedCiliumPolicy(t.Context(), testNamespace, testK8sName)
			if strings.HasSuffix(state, "unavailable") {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			_, err = policies.Resource(cilium.NetworkPolicyResource).Namespace(testNamespace).Get(t.Context(), policy.GetName(), metav1.GetOptions{})
			if state == "empty" {
				require.True(t, apierrors.IsNotFound(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func policyTestClient(objects ...runtime.Object) *fakedynamic.FakeDynamicClient {
	client := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		cilium.NetworkPolicyResource: "CiliumNetworkPolicyList",
	}, objects...)
	client.PrependReactor("patch", "ciliumnetworkpolicies", func(action clienttesting.Action) (bool, runtime.Object, error) {
		policy := new(unstructured.Unstructured)
		if err := json.Unmarshal(action.(clienttesting.PatchAction).GetPatch(), policy); err != nil {
			return true, nil, err
		}
		_, err := client.Tracker().Get(cilium.NetworkPolicyResource, policy.GetNamespace(), policy.GetName())
		if apierrors.IsNotFound(err) {
			err = client.Tracker().Create(cilium.NetworkPolicyResource, policy, policy.GetNamespace())
		} else if err == nil {
			err = client.Tracker().Update(cilium.NetworkPolicyResource, policy, policy.GetNamespace())
		}
		return true, policy, err
	})
	return client
}

func TestCiliumPolicyGrantsScopedPrivateNetworkSelfTraffic(t *testing.T) {
	for _, kind := range []string{"production", "preview"} {
		t.Run(kind, func(t *testing.T) {
			policy := applyPolicy(t, true, kind, "10.0.0.53")

			selector, found, err := unstructured.NestedStringMap(policy.Object, "spec", "endpointSelector", "matchLabels")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, map[string]string{
				labels.LabelKeyWorkspaceID:  testWorkspaceID,
				labels.LabelKeyProjectID:    testProjectID,
				labels.LabelKeyAppID:        testAppID,
				labels.LabelKeyDeploymentID: testDeploymentID,
			}, selector)

			defaultDeny, found, err := unstructured.NestedMap(policy.Object, "spec", "enableDefaultDeny")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, map[string]interface{}{"ingress": true, "egress": false}, defaultDeny)

			ingress, found, err := unstructured.NestedSlice(policy.Object, "spec", "ingress")
			require.NoError(t, err)
			require.True(t, found)
			require.Len(t, ingress, 2)
			assertSelfRule(t, ingress[1].(map[string]interface{}), "fromEndpoints")

			egress, found, err := unstructured.NestedSlice(policy.Object, "spec", "egress")
			require.NoError(t, err)
			require.True(t, found)
			require.Len(t, egress, 1, "no broad egress rule may accompany the scoped self grant")
			assertSelfRule(t, egress[0].(map[string]interface{}), "toEndpoints")
		})
	}
}

func TestCiliumPolicyDoesNotGrantUnenrolledDeployment(t *testing.T) {
	policy := applyPolicy(t, false, "preview", "10.0.0.53")
	_, found, err := unstructured.NestedMap(policy.Object, "spec", "enableDefaultDeny")
	require.NoError(t, err)
	require.False(t, found)
	_, found, err = unstructured.NestedSlice(policy.Object, "spec", "egress")
	require.NoError(t, err)
	require.False(t, found)
	ingress, found, err := unstructured.NestedSlice(policy.Object, "spec", "ingress")
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, ingress, 1)
}

func TestCiliumPolicyDoesNotGrantWithoutRegionalResolver(t *testing.T) {
	policy := applyPolicy(t, true, "preview", "")
	_, found, err := unstructured.NestedSlice(policy.Object, "spec", "egress")
	require.NoError(t, err)
	require.False(t, found)
}

func applyPolicy(t *testing.T, private bool, kind, resolver string) *unstructured.Unstructured {
	t.Helper()
	client := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	var policy *unstructured.Unstructured
	client.PrependReactor("patch", "ciliumnetworkpolicies", func(action clienttesting.Action) (bool, runtime.Object, error) {
		policy = new(unstructured.Unstructured)
		err := json.Unmarshal(action.(clienttesting.PatchAction).GetPatch(), policy)
		return true, policy, err
	})
	controller := &Controller{dynamicClient: client, privateNetworkResolverIP: resolver}
	req := fullApplyRequest(t)
	req.EnvironmentKind = kind
	req.PrivateNetworking = private
	if private {
		req.PrivateNetworkReplicaHost = "api.unkey.internal"
	}
	require.NoError(t, controller.ensureCiliumNetworkPolicy(t.Context(), req, nil))
	require.NotNil(t, policy)
	return policy
}

func assertSelfRule(t *testing.T, rule map[string]interface{}, endpointKey string) {
	t.Helper()
	endpoints := rule[endpointKey].([]interface{})
	require.Len(t, endpoints, 1)
	endpoint := endpoints[0].(map[string]interface{})
	require.Equal(t, map[string]interface{}{
		labels.LabelKeyManagedBy:    "krane",
		labels.LabelKeyComponent:    "deployment",
		labels.LabelKeyWorkspaceID:  testWorkspaceID,
		labels.LabelKeyProjectID:    testProjectID,
		labels.LabelKeyAppID:        testAppID,
		labels.LabelKeyDeploymentID: testDeploymentID,
	}, endpoint["matchLabels"])
	require.ElementsMatch(t, []interface{}{
		map[string]interface{}{"key": labels.LabelKeyNamespace, "operator": "Exists"},
		map[string]interface{}{"key": "io.cilium.k8s.policy.cluster", "operator": "Exists"},
	}, endpoint["matchExpressions"])
	require.Equal(t, []interface{}{map[string]interface{}{"ports": []interface{}{
		map[string]interface{}{"port": "1", "endPort": int64(65535), "protocol": "TCP"},
		map[string]interface{}{"port": "1", "endPort": int64(65535), "protocol": "UDP"},
	}}}, rule["toPorts"])
}
