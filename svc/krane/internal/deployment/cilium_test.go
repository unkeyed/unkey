package deployment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestCiliumPolicyGrantsScopedPrivateNetworkSelfTraffic(t *testing.T) {
	policy := applyPolicy(t, true, true)

	selector, found, err := unstructured.NestedStringMap(policy.Object, "spec", "endpointSelector", "matchLabels")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, map[string]string{labels.LabelKeyDeploymentID: testDeploymentID}, selector)

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
}

func TestCiliumPolicyDoesNotGrantUnenrolledDeployment(t *testing.T) {
	policy := applyPolicy(t, false, true)
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

func TestCiliumPolicyDoesNotGrantWhenPrivateNetworkDisabled(t *testing.T) {
	policy := applyPolicy(t, true, false)
	_, found, err := unstructured.NestedSlice(policy.Object, "spec", "egress")
	require.NoError(t, err)
	require.False(t, found)
}

func applyPolicy(t *testing.T, private, clusterEnabled bool) *unstructured.Unstructured {
	t.Helper()
	client := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	var policy *unstructured.Unstructured
	client.PrependReactor("patch", "ciliumnetworkpolicies", func(action clienttesting.Action) (bool, runtime.Object, error) {
		policy = new(unstructured.Unstructured)
		err := json.Unmarshal(action.(clienttesting.PatchAction).GetPatch(), policy)
		return true, policy, err
	})
	controller := &Controller{dynamicClient: client, privateNetworkingEnabled: clusterEnabled}
	req := fullApplyRequest(t)
	req.Capabilities = &ctrlv1.DeploymentCapabilities{PrivateNetworking: private}
	require.NoError(t, controller.ensureCiliumNetworkPolicy(t.Context(), req, metav1.OwnerReference{}))
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
