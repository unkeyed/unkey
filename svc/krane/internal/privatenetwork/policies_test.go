package privatenetwork

import (
	"cmp"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
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

func TestBindingPolicyGrantsEveryUnicastPortInOneDirection(t *testing.T) {
	app := testApp("target_1")
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	name := policyName(app)
	require.NoError(t, r.ensurePolicy(t.Context(), app, name, nil))

	policy, err := dynamic.Resource(policyResource).Namespace(app.GetK8SNamespace()).Get(t.Context(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotContains(t, policy.GetAnnotations(), "bindings.unkey.com/port")
	specs := policySpecs(t, policy)
	require.Len(t, specs, 2)

	caller, target := specs[0], specs[1]
	require.Contains(t, caller, "egress")
	require.NotContains(t, caller, "ingress")
	require.Contains(t, target, "ingress")
	require.NotContains(t, target, "egress")
	require.Equal(t, app.GetCallerDeploymentId(), selectorLabels(t, caller)[labels.LabelKeyDeploymentID])
	require.Equal(t, app.GetDeploymentId(), selectorLabels(t, target)[labels.LabelKeyDeploymentID])
	require.NotContains(t, selectorLabels(t, caller), labels.LabelKeyAppID)
	require.Equal(t, app.GetAppId(), selectorLabels(t, target)[labels.LabelKeyAppID])

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

	require.Equal(t, []flow{{"caller_1", "target_1"}}, effectiveFlows(t, dynamic, app.GetK8SNamespace()))
}

func TestBindingPolicyReverseRequiresReciprocalBinding(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	forward := testApp("dep_b")
	forward.CallerDeploymentId = "dep_a"
	require.NoError(t, r.ensurePolicy(t.Context(), forward, policyName(forward), nil))
	require.Equal(t, []flow{{"dep_a", "dep_b"}}, effectiveFlows(t, dynamic, forward.GetK8SNamespace()))

	reverse := testApp("dep_a")
	reverse.AppId, reverse.BindingId, reverse.CallerDeploymentId = "app_a", "binding_reverse", "dep_b"
	require.NoError(t, r.ensurePolicy(t.Context(), reverse, policyName(reverse), nil))
	require.ElementsMatch(t, []flow{{"dep_a", "dep_b"}, {"dep_b", "dep_a"}}, effectiveFlows(t, dynamic, forward.GetK8SNamespace()))
}

func TestBindingPolicyIsNotTransitiveAndExcludesUnrelatedDeployments(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	aToB := testApp("dep_b")
	aToB.CallerDeploymentId, aToB.BindingId = "dep_a", "binding_ab"
	bToC := testApp("dep_c")
	bToC.AppId, bToC.CallerDeploymentId, bToC.BindingId = "app_c", "dep_b", "binding_bc"
	for _, app := range []*ctrlv1.PrivateNetworkApp{aToB, bToC} {
		require.NoError(t, r.ensurePolicy(t.Context(), app, policyName(app), nil))
	}

	flows := effectiveFlows(t, dynamic, aToB.GetK8SNamespace())
	require.ElementsMatch(t, []flow{{"dep_a", "dep_b"}, {"dep_b", "dep_c"}}, flows)
	for _, forbidden := range []flow{{"dep_a", "dep_c"}, {"dep_c", "dep_a"}, {"dep_b", "dep_a"}, {"dep_c", "dep_b"}, {"dep_unrelated", "dep_b"}} {
		require.NotContains(t, flows, forbidden)
	}
}

func TestSelfBindingPeersOnlyReplicasOfCallerDeployment(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	self := testApp("caller_1")
	self.AppId = "app_caller"
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

func TestBindingPolicyIsolationAndImmediateRevocation(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	first := testApp("target_1")
	second := testApp("target_1")
	second.CallerDeploymentId, second.BindingId = "caller_2", "binding_2"
	for _, app := range []*ctrlv1.PrivateNetworkApp{first, second} {
		require.NoError(t, r.ensurePolicy(t.Context(), app, policyName(app), nil))
	}
	policies, err := dynamic.Resource(policyResource).Namespace(first.GetK8SNamespace()).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies.Items, 2)

	first.DeploymentId, first.Port = "", 0
	require.NoError(t, r.ensurePolicy(t.Context(), first, policyName(first), nil))
	_, err = dynamic.Resource(policyResource).Namespace(first.GetK8SNamespace()).Get(t.Context(), policyName(first), metav1.GetOptions{})
	require.Error(t, err)
	require.Equal(t, []flow{{"caller_2", "target_1"}}, effectiveFlows(t, dynamic, first.GetK8SNamespace()))
}

func TestBindingPolicyIgnoresTargetPortChanges(t *testing.T) {
	dynamic := testDynamicClient()
	r := &Reconciler{dynamic: dynamic}
	app := testApp("target_1")
	require.NoError(t, r.ensurePolicy(t.Context(), app, policyName(app), nil))
	before, err := dynamic.Resource(policyResource).Namespace(app.GetK8SNamespace()).Get(t.Context(), policyName(app), metav1.GetOptions{})
	require.NoError(t, err)

	app.Port = 9090
	require.NoError(t, r.ensurePolicy(t.Context(), app, policyName(app), nil))
	after, err := dynamic.Resource(policyResource).Namespace(app.GetK8SNamespace()).Get(t.Context(), policyName(app), metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, before.GetResourceVersion(), after.GetResourceVersion(), "a declared port change must not rewrite access")
}

func policyName(app *ctrlv1.PrivateNetworkApp) string {
	return resourceName("unkey-pn-binding", app.GetBindingId()+"/"+app.GetCallerDeploymentId())
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
