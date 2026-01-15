package reconciler

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
	NamespaceSentinel = "sentinel"
	SentinelNodeClass = "sentinel"
	CustomerNodeClass = "untrusted"
	SentinelPort      = 8040
	DeploymentPort    = 8080
)

// ensureNamespaceExists creates the namespace if it doesn't already exist.
// AlreadyExists errors are treated as success.
// For customer namespaces (non-sentinel), it also creates a CiliumNetworkPolicy
// to allow ingress only from the matching sentinel.
func (r *Reconciler) ensureNamespaceExists(ctx context.Context, namespace, workspaceID, environmentID string) error {
	_, err := r.clientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// Create CiliumNetworkPolicy for customer namespaces (not for sentinel namespace)
	if namespace != NamespaceSentinel {
		if err := r.applyCiliumPolicyForNamespace(ctx, namespace, workspaceID, environmentID); err != nil {
			return fmt.Errorf("failed to create cilium policy for namespace %s: %w", namespace, err)
		}
	}

	return nil
}

// applyCiliumPolicyForNamespace creates a CiliumNetworkPolicy that allows ingress
// only from sentinels with matching workspace and environment IDs.
func (r *Reconciler) applyCiliumPolicyForNamespace(ctx context.Context, namespace, workspaceID, environmentID string) error {
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

	_, err = r.dynamicClient.Resource(gvr).Namespace(namespace).Apply(
		ctx,
		"allow-sentinel-ingress",
		unstructuredPolicy,
		metav1.ApplyOptions{FieldManager: "krane"},
	)
	return err
}

// toUnstructured converts a typed Kubernetes object to an unstructured object.
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
