package privatenetwork

import (
	"cmp"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/appconnection"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

type flow struct {
	from, to string
}

type portProto struct {
	port     int
	protocol string
}

func TestConnectionPolicyGrantsEveryUnicastPortInOneDirection(t *testing.T) {
	connectionSpec := testConnection("target_1")
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	name := policyName(connectionSpec)
	require.NoError(t, r.ensurePolicy(t.Context(), connectionSpec, name, nil))

	policy, err := dynamic.Resource(policyResource).Namespace(connectionSpec.GetK8SNamespace()).Get(t.Context(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotContains(t, policy.GetAnnotations(), "connections.unkey.com/port")
	specs := policySpecs(t, policy)
	require.Len(t, specs, 2)

	caller, target := specs[0], specs[1]
	require.Contains(t, caller, "egress")
	require.NotContains(t, caller, "ingress")
	require.Contains(t, target, "ingress")
	require.NotContains(t, target, "egress")
	require.Equal(t, connectionSpec.GetCallerDeploymentId(), selectorLabels(t, caller)[labels.LabelKeyDeploymentID])
	require.Equal(t, connectionSpec.GetTargetDeploymentId(), selectorLabels(t, target)[labels.LabelKeyDeploymentID])
	require.NotContains(t, selectorLabels(t, caller), labels.LabelKeyAppID)
	require.Equal(t, connectionSpec.GetTargetAppId(), selectorLabels(t, target)[labels.LabelKeyAppID])

	for _, spec := range []map[string]interface{}{caller, target} {
		ruleKey := "egress"
		if _, ok := spec["ingress"]; ok {
			ruleKey = "ingress"
		}

		ports := rulePorts(t, spec, ruleKey)
		for _, allowed := range []portProto{
			{8080, "TCP"}, {8081, "TCP"}, {7946, "TCP"}, {1, "TCP"}, {65535, "TCP"},
			{7946, "UDP"}, {8301, "UDP"}, {53, "UDP"}, {65535, "UDP"},
		} {
			require.True(t, allowsPort(ports, allowed), "%s %s/%d must be allowed", ruleKey, allowed.protocol, allowed.port)
		}
		for _, denied := range []portProto{{0, "TCP"}, {8080, "SCTP"}, {0, "ICMP"}} {
			require.False(t, allowsPort(ports, denied), "%s %s/%d must stay denied", ruleKey, denied.protocol, denied.port)
		}

		expressions := spec["endpointSelector"].(map[string]interface{})["matchExpressions"].([]interface{})
		require.ElementsMatch(t, []interface{}{
			map[string]interface{}{"key": labels.LabelKeyNamespace, "operator": "Exists"},
			map[string]interface{}{"key": "io.cilium.k8s.policy.cluster", "operator": "Exists"},
		}, expressions)
	}

	require.Equal(t, []flow{{"caller_1", "target_1"}}, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()))
}

func TestConnectionPolicyReverseRequiresReciprocalConnection(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	forward := testConnection("dep_b")
	forward.CallerDeploymentId = "dep_a"
	require.NoError(t, r.ensurePolicy(t.Context(), forward, policyName(forward), nil))
	require.Equal(t, []flow{{"dep_a", "dep_b"}}, effectiveFlows(t, dynamic, forward.GetK8SNamespace()))

	reverse := testConnection("dep_a")
	reverse.TargetAppId, reverse.ConnectionId, reverse.CallerDeploymentId = "app_a", "connection_reverse", "dep_b"
	require.NoError(t, r.ensurePolicy(t.Context(), reverse, policyName(reverse), nil))
	require.ElementsMatch(t, []flow{{"dep_a", "dep_b"}, {"dep_b", "dep_a"}}, effectiveFlows(t, dynamic, forward.GetK8SNamespace()))
}

func TestConnectionPolicyIsNotTransitiveAndExcludesUnrelatedDeployments(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	aToB := testConnection("dep_b")
	aToB.CallerDeploymentId, aToB.ConnectionId = "dep_a", "connection_ab"
	bToC := testConnection("dep_c")
	bToC.TargetAppId, bToC.CallerDeploymentId, bToC.ConnectionId = "app_c", "dep_b", "connection_bc"
	for _, connectionSpec := range []*ctrlv1.PrivateNetworkConnection{aToB, bToC} {
		require.NoError(t, r.ensurePolicy(t.Context(), connectionSpec, policyName(connectionSpec), nil))
	}

	flows := effectiveFlows(t, dynamic, aToB.GetK8SNamespace())
	require.ElementsMatch(t, []flow{{"dep_a", "dep_b"}, {"dep_b", "dep_c"}}, flows)
	for _, forbidden := range []flow{{"dep_a", "dep_c"}, {"dep_c", "dep_a"}, {"dep_b", "dep_a"}, {"dep_c", "dep_b"}, {"dep_unrelated", "dep_b"}} {
		require.NotContains(t, flows, forbidden)
	}
}

func TestSelfConnectionPeersOnlyReplicasOfCallerDeployment(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	self := testConnection("caller_1")
	self.TargetAppId = "app_caller"
	require.NoError(t, r.ensurePolicy(t.Context(), self, policyName(self), nil))

	require.Equal(t, []flow{{"caller_1", "caller_1"}}, effectiveFlows(t, dynamic, self.GetK8SNamespace()))
	policy, err := dynamic.Resource(policyResource).Namespace(self.GetK8SNamespace()).Get(t.Context(), policyName(self), metav1.GetOptions{})
	require.NoError(t, err)
	for _, spec := range policySpecs(t, policy) {
		selector := selectorLabels(t, spec)
		require.Equal(t, "ws_1", selector[labels.LabelKeyWorkspaceID])
		require.Equal(t, "proj_1", selector[labels.LabelKeyProjectID])
		require.Equal(t, "caller_1", selector[labels.LabelKeyDeploymentID])
		require.NotContains(t, selector, labels.LabelKeyEnvironmentID, "environment isolation comes from the deployment ID, not a broader selector")
		require.NotContains(t, selector, "io.cilium.k8s.policy.cluster", "replicas in every region must match")
	}
}

func TestConnectionPolicyObjectsAreIsolatedAndRemovedOnRevocation(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	first := testConnection("target_1")
	second := testConnection("target_1")
	second.CallerDeploymentId, second.ConnectionId = "caller_2", "connection_2"
	for _, connectionSpec := range []*ctrlv1.PrivateNetworkConnection{first, second} {
		require.NoError(t, r.ensurePolicy(t.Context(), connectionSpec, policyName(connectionSpec), nil))
	}
	policies, err := dynamic.Resource(policyResource).Namespace(first.GetK8SNamespace()).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies.Items, 2)

	first.TargetDeploymentId, first.TargetPort = "", 0
	require.NoError(t, r.ensurePolicy(t.Context(), first, policyName(first), nil))
	_, err = dynamic.Resource(policyResource).Namespace(first.GetK8SNamespace()).Get(t.Context(), policyName(first), metav1.GetOptions{})
	require.Error(t, err)
	require.Equal(t, []flow{{"caller_2", "target_1"}}, effectiveFlows(t, dynamic, first.GetK8SNamespace()))
}

func TestConnectionPolicyRetainsOldTargetUntilReplacementCoverageExpires(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic, now: func() time.Time { return now }}
	connectionSpec := testConnection("dep_a")
	name := policyName(connectionSpec)
	require.NoError(t, r.ensurePolicy(ctx, connectionSpec, name, nil))

	connectionSpec.TargetDeploymentId = "dep_b"
	published := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Labels: connectionLabels(connectionSpec)}, Data: map[string]string{
		"appSlug": connectionSpec.GetConnectionName(), "deploymentId": "dep_a",
	}}
	require.NoError(t, r.ensurePolicy(ctx, connectionSpec, name, published))
	require.ElementsMatch(t, []flow{{"caller_1", "dep_a"}, {"caller_1", "dep_b"}}, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()))

	now = now.Add(appconnection.ReplacementOverlap)
	require.NoError(t, r.ensurePolicy(ctx, connectionSpec, name, published))
	require.Contains(t, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()), flow{"caller_1", "dep_a"}, "the published old target still requires coverage")

	published.Data["deploymentId"] = "dep_b"
	require.NoError(t, r.ensurePolicy(ctx, connectionSpec, name, published))
	now = now.Add(appconnection.ReplacementOverlap - time.Nanosecond)
	require.NoError(t, r.ensurePolicy(ctx, connectionSpec, name, published))
	require.Contains(t, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()), flow{"caller_1", "dep_a"})
	now = now.Add(time.Nanosecond)
	require.NoError(t, r.ensurePolicy(ctx, connectionSpec, name, published))
	require.Equal(t, []flow{{"caller_1", "dep_b"}}, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()))
}

func TestConnectionPolicyIgnoresTargetPortChanges(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	connectionSpec := testConnection("target_1")
	require.NoError(t, r.ensurePolicy(t.Context(), connectionSpec, policyName(connectionSpec), nil))
	before, err := dynamic.Resource(policyResource).Namespace(connectionSpec.GetK8SNamespace()).Get(t.Context(), policyName(connectionSpec), metav1.GetOptions{})
	require.NoError(t, err)

	connectionSpec.TargetPort = 9090
	require.NoError(t, r.ensurePolicy(t.Context(), connectionSpec, policyName(connectionSpec), nil))
	after, err := dynamic.Resource(policyResource).Namespace(connectionSpec.GetK8SNamespace()).Get(t.Context(), policyName(connectionSpec), metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, before.GetResourceVersion(), after.GetResourceVersion(), "a declared port change must not rewrite access")
}

func policyName(connectionSpec *ctrlv1.PrivateNetworkConnection) string {
	return resourceName("unkey-pn-connection", connectionSpec.GetConnectionId()+"/"+connectionSpec.GetCallerDeploymentId())
}

func policySpecs(t *testing.T, policy *unstructured.Unstructured) []map[string]interface{} {
	t.Helper()
	raw, found, err := unstructured.NestedSlice(policy.Object, "specs")
	require.NoError(t, err)
	require.True(t, found)
	specs := make([]map[string]interface{}, 0, len(raw))
	for _, spec := range raw {
		specs = append(specs, spec.(map[string]interface{}))
	}
	return specs
}

func rulePorts(t *testing.T, spec map[string]interface{}, ruleKey string) []interface{} {
	t.Helper()
	rules := spec[ruleKey].([]interface{})
	require.Len(t, rules, 1)
	toPorts := rules[0].(map[string]interface{})["toPorts"].([]interface{})
	require.Len(t, toPorts, 1)
	return toPorts[0].(map[string]interface{})["ports"].([]interface{})
}

func allowsPort(ports []interface{}, want portProto) bool {
	for _, raw := range ports {
		entry := raw.(map[string]interface{})
		if entry["protocol"] != want.protocol {
			continue
		}
		start, err := strconv.Atoi(entry["port"].(string))
		if err != nil {
			continue
		}
		end := start
		if endPort, ok := entry["endPort"].(int64); ok {
			end = int(endPort)
		}
		if start >= 1 && want.port >= start && want.port <= end {
			return true
		}
	}
	return false
}

// effectiveFlows returns caller-to-target deployment pairs that both an egress
// rule on the source and an ingress rule on the destination admit, as Cilium
// default deny requires.
func effectiveFlows(t *testing.T, dynamic *fakedynamic.FakeDynamicClient, namespace string) []flow {
	t.Helper()
	policies, err := dynamic.Resource(policyResource).Namespace(namespace).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)

	egress := map[flow]bool{}
	ingress := map[flow]bool{}
	for i := range policies.Items {
		for _, spec := range policySpecs(t, &policies.Items[i]) {
			self := selectorLabels(t, spec)[labels.LabelKeyDeploymentID].(string)
			for _, direction := range []struct {
				rule, peers string
			}{{"egress", "toEndpoints"}, {"ingress", "fromEndpoints"}} {
				rules, ok := spec[direction.rule].([]interface{})
				if !ok {
					continue
				}
				for _, rule := range rules {
					for _, peer := range rule.(map[string]interface{})[direction.peers].([]interface{}) {
						other := peer.(map[string]interface{})["matchLabels"].(map[string]interface{})[labels.LabelKeyDeploymentID].(string)
						if direction.rule == "egress" {
							egress[flow{self, other}] = true
						} else {
							ingress[flow{other, self}] = true
						}
					}
				}
			}
		}
	}

	var flows []flow
	for f := range egress {
		if ingress[f] {
			flows = append(flows, f)
		}
	}

	slices.SortFunc(flows, func(a, b flow) int {
		return cmp.Or(cmp.Compare(a.from, b.from), cmp.Compare(a.to, b.to))
	})
	return flows
}

func selectorLabels(t *testing.T, spec map[string]interface{}) map[string]interface{} {
	t.Helper()
	selector := spec["endpointSelector"].(map[string]interface{})
	return selector["matchLabels"].(map[string]interface{})
}
