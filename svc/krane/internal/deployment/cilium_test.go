package deployment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestCiliumPolicyOnlyAllowsFrontlineIngress(t *testing.T) {
	for _, tc := range []struct {
		name, kind string
	}{
		{"production", "production"},
		{"preview", "preview"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
			var policy *unstructured.Unstructured
			client.PrependReactor("patch", "ciliumnetworkpolicies", func(action clienttesting.Action) (bool, runtime.Object, error) {
				policy = new(unstructured.Unstructured)
				err := json.Unmarshal(action.(clienttesting.PatchAction).GetPatch(), policy)
				return true, policy, err
			})
			controller := &Controller{dynamicClient: client, privateNetworkResolverIP: "10.0.0.53"}
			req := fullApplyRequest(t)
			req.EnvironmentKind = tc.kind

			require.NoError(t, controller.ensureCiliumNetworkPolicy(t.Context(), req, nil))
			rules, found, err := unstructured.NestedSlice(policy.Object, "spec", "ingress")
			require.NoError(t, err)
			require.True(t, found)
			require.Len(t, rules, 1)

			_, found, err = unstructured.NestedSlice(policy.Object, "spec", "egress")
			require.NoError(t, err)
			require.False(t, found, "deployment policy must not grant implicit peer egress")
		})
	}
}
