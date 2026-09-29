package privatenetwork

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/appbinding"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const policyTargetsAnnotation = "bindings.unkey.com/targets"

var policyResource = schema.GroupVersionResource{Group: "cilium.io", Version: "v2", Resource: "ciliumnetworkpolicies"}

func (r *Reconciler) ensurePolicy(ctx context.Context, app *ctrlv1.PrivateNetworkApp, name string, binding *corev1.ConfigMap) error {
	client := r.dynamic.Resource(policyResource).Namespace(app.GetK8SNamespace())
	current, err := client.Get(ctx, name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("get binding policy %s: %w", name, err)
	}
	if apierrors.IsNotFound(err) {
		current = nil
	}

	targets := make(map[string]time.Time)
	if current != nil {
		l := current.GetLabels()
		if !owned(l) || l[labels.LabelKeyWorkspaceID] != app.GetWorkspaceId() ||
			l[labels.LabelKeyProjectID] != app.GetProjectId() || l[labels.LabelKeyBindingID] != app.GetBindingId() ||
			l[labels.LabelKeyCallerDeploymentID] != app.GetCallerDeploymentId() {
			return fmt.Errorf("refuse to replace foreign binding policy %s", name)
		}
		if maps.Equal(l, bindingLabels(app)) && app.GetDeploymentId() != "" {
			if err := json.Unmarshal([]byte(current.GetAnnotations()[policyTargetsAnnotation]), &targets); err != nil {
				return fmt.Errorf("read binding policy targets %s: %w", name, err)
			}
		}
	}

	if app.GetDeploymentId() != "" {
		active := ""
		if binding != nil && maps.Equal(binding.Labels, bindingLabels(app)) && binding.Data["appSlug"] == app.GetBindingName() {
			active = binding.Data["deploymentId"]
		}
		for target, deadline := range targets {
			if target == app.GetDeploymentId() || target == active {
				continue
			}
			if deadline.IsZero() {
				targets[target] = r.clock().Add(appbinding.ReplacementOverlap)
			} else if !r.clock().Before(deadline) {
				delete(targets, target)
			}
		}
		targets[app.GetDeploymentId()] = time.Time{}
		if active != "" {
			targets[active] = time.Time{}
		}
	}

	encoded, err := json.Marshal(targets)
	if err != nil {
		return fmt.Errorf("encode binding policy targets: %w", err)
	}

	specs := make([]interface{}, 0, 2*len(targets))
	for _, target := range slices.Sorted(maps.Keys(targets)) {
		caller := bindingEndpoint(app, app.GetCallerDeploymentId(), false)
		peer := bindingEndpoint(app, target, true)
		ports := unicastPorts()
		specs = append(specs,
			map[string]interface{}{
				"endpointSelector":  caller,
				"enableDefaultDeny": map[string]interface{}{"ingress": false, "egress": false},
				"egress":            []interface{}{map[string]interface{}{"toEndpoints": []interface{}{peer}, "toPorts": ports}},
			},
			map[string]interface{}{
				"endpointSelector":  peer,
				"enableDefaultDeny": map[string]interface{}{"ingress": false, "egress": false},
				"ingress":           []interface{}{map[string]interface{}{"fromEndpoints": []interface{}{caller}, "toPorts": ports}},
			},
		)
	}

	if len(specs) == 0 {
		if current == nil {
			return nil
		}
		if err := client.Delete(ctx, name, deleteOptions(current)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("revoke unresolved binding policy %s: %w", name, err)
		}
		return nil
	}

	desired := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "cilium.io/v2", "kind": "CiliumNetworkPolicy", "specs": specs,
	}}
	desired.SetName(name)
	desired.SetNamespace(app.GetK8SNamespace())
	desired.SetLabels(bindingLabels(app))
	desired.SetAnnotations(map[string]string{policyTargetsAnnotation: string(encoded)})

	if current != nil {
		if maps.Equal(current.GetLabels(), desired.GetLabels()) && maps.Equal(current.GetAnnotations(), desired.GetAnnotations()) &&
			equality.Semantic.DeepEqual(current.Object["specs"], desired.Object["specs"]) {
			return nil
		}
		desired.SetResourceVersion(current.GetResourceVersion())
		_, err = client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager})
	} else {
		_, err = client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	}
	if err != nil {
		return fmt.Errorf("publish binding policy %s: %w", name, err)
	}
	return nil
}

// unicastPorts allows every TCP and UDP port. An explicit range keeps ICMP,
// SCTP, and non-port protocols denied, which an omitted toPorts would allow.
func unicastPorts() []interface{} {
	return []interface{}{map[string]interface{}{"ports": []interface{}{
		map[string]interface{}{"port": "1", "endPort": int64(65535), "protocol": "TCP"},
		map[string]interface{}{"port": "1", "endPort": int64(65535), "protocol": "UDP"},
	}}}
}

func bindingEndpoint(app *ctrlv1.PrivateNetworkApp, deployment string, target bool) map[string]interface{} {
	l := map[string]interface{}{
		labels.LabelKeyWorkspaceID: app.GetWorkspaceId(), labels.LabelKeyProjectID: app.GetProjectId(),
		labels.LabelKeyDeploymentID: deployment, labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: "deployment",
	}
	if target {
		l[labels.LabelKeyAppID] = app.GetAppId()
	}
	return map[string]interface{}{
		"matchLabels": l,
		"matchExpressions": []interface{}{
			map[string]interface{}{"key": labels.LabelKeyNamespace, "operator": "Exists"},
			map[string]interface{}{"key": "io.cilium.k8s.policy.cluster", "operator": "Exists"},
		},
	}
}

func (r *Reconciler) cleanupPolicies(ctx context.Context, desired map[string]struct{}) error {
	policies, err := r.dynamic.Resource(policyResource).Namespace("").List(ctx, metav1.ListOptions{
		LabelSelector: labels.LabelKeyManagedBy + "=krane," + labels.LabelKeyComponent + "=" + component,
	})
	if err != nil {
		return fmt.Errorf("list binding policies: %w", err)
	}
	for i := range policies.Items {
		policy := &policies.Items[i]
		if _, ok := desired[policy.GetNamespace()+"/"+policy.GetName()]; ok || !owned(policy.GetLabels()) {
			continue
		}
		if err := r.dynamic.Resource(policyResource).Namespace(policy.GetNamespace()).Delete(ctx, policy.GetName(), deleteOptions(policy)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete binding policy: %w", err)
		}
	}
	return nil
}
