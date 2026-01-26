package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	slim_metav1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cilium/cilium/pkg/policy/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// NamespaceSentinel is the namespace where all sentinel pods run, separate
	// from customer deployment namespaces for isolation.
	NamespaceSentinel = "sentinel"
)

// ensureNamespaceExists creates the namespace if it doesn't already exist and
// configures network policies for customer workloads.
//
// For customer namespaces (anything except "sentinel"), the method also applies
// a CiliumNetworkPolicy that restricts ingress to only sentinels with matching
// workspace and environment IDs. This ensures customer code can only be reached
// by its designated sentinel, not by other tenants' workloads.
//
// The method is idempotent: calling it for an existing namespace succeeds without error.
func (c *Controller) ensureNamespaceExists(ctx context.Context, namespace, workspaceID, environmentID string) error {
	_, err := c.clientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	if namespace != NamespaceSentinel {
		if err := c.applyCiliumPolicyForNamespace(ctx, namespace, workspaceID, environmentID); err != nil {
			return fmt.Errorf("failed to create cilium policy for namespace %s: %w", namespace, err)
		}
	}

	return nil
}

// applyCiliumPolicyForNamespace creates or updates a CiliumNetworkPolicy that
// restricts ingress to pods matching specific labels.
//
// The policy allows TCP traffic on [DeploymentPort] only from pods in the "sentinel"
// namespace with matching workspace and environment ID labels. This isolates each
// tenant's deployments from other tenants' sentinels and from direct external access.
//
// The method uses server-side apply, making it safe to call repeatedly.
//
//nolint:exhaustruct
func (c *Controller) applyCiliumPolicyForNamespace(ctx context.Context, namespace, workspaceID, environmentID string) error {
	policy := &ciliumv2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cilium.io/v2",
			Kind:       "CiliumNetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-sentinel-ingress",
			Namespace: namespace,
		},
		Spec: &api.Rule{
			Description: fmt.Sprintf("Allow ingress from sentinel for workspace %s environment %s", workspaceID, environmentID),
			EndpointSelector: api.EndpointSelector{
				LabelSelector: &slim_metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/managed-by": "krane",
						"app.kubernetes.io/component":  "deployment",
					},
				},
			},
			Ingress: []api.IngressRule{
				{
					IngressCommonRule: api.IngressCommonRule{
						FromEndpoints: []api.EndpointSelector{
							{
								LabelSelector: &slim_metav1.LabelSelector{
									MatchLabels: map[string]string{
										"io.kubernetes.pod.namespace": NamespaceSentinel,
										"unkey.com/workspace.id":      workspaceID,
										"unkey.com/environment.id":    environmentID,
									},
								},
							},
						},
					},
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     strconv.Itoa(DeploymentPort),
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	unstructuredPolicy, err := toUnstructured(policy)
	if err != nil {
		return fmt.Errorf("failed to convert policy to unstructured: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumnetworkpolicies",
	}

	_, err = c.dynamicClient.Resource(gvr).Namespace(namespace).Apply(
		ctx,
		"allow-sentinel-ingress",
		unstructuredPolicy,
		metav1.ApplyOptions{FieldManager: "krane"},
	)
	return err
}

// toUnstructured converts a typed Kubernetes object to an unstructured representation
// for use with the dynamic client. This is needed for CRDs like CiliumNetworkPolicy
// that may not have generated client types available.
func toUnstructured(obj any) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	u := &unstructured.Unstructured{}
	if err := json.Unmarshal(data, &u.Object); err != nil {
		return nil, err
	}
	return u, nil
}
