package privatenetwork

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const policyTargetsAnnotation = "connections.unkey.com/targets"

func (r *Reconciler) ensurePolicy(ctx context.Context, connectionSpec *ctrlv1.PrivateNetworkConnection, name string, targetsActive []string, current *unstructured.Unstructured) error {
	client := r.dynamic.Resource(cilium.NetworkPolicyResource).Namespace(connectionSpec.GetK8SNamespace())
	labelsDesired := connectionLabels(connectionSpec)

	targets := make(map[string]time.Time)
	if current != nil {
		if !ownedByConnection(current.GetLabels(), connectionSpec) {
			return fmt.Errorf("refuse to replace foreign connection policy %s", name)
		}
		if maps.Equal(current.GetLabels(), labelsDesired) && connectionSpec.GetTargetDeploymentId() != "" {
			if err := json.Unmarshal([]byte(current.GetAnnotations()[policyTargetsAnnotation]), &targets); err != nil {
				return fmt.Errorf("read connection policy targets %s: %w", name, err)
			}
		}
	}

	if connectionSpec.GetTargetDeploymentId() != "" {
		now := r.clock.Now()
		for target, deadline := range targets {
			if slices.Contains(targetsActive, target) {
				continue
			}
			if deadline.IsZero() {
				targets[target] = now.Add(privatenetwork.ReplacementOverlap)
			} else if !now.Before(deadline) {
				delete(targets, target)
			}
		}
		for _, target := range targetsActive {
			targets[target] = time.Time{}
		}
	}

	encoded, err := json.Marshal(targets)
	if err != nil {
		return fmt.Errorf("encode connection policy targets: %w", err)
	}

	caller := connectionEndpoint(connectionSpec, connectionSpec.GetCallerDeploymentId(), false)
	ports := cilium.UnicastPorts()
	specs := make([]interface{}, 0, 2*len(targets))
	for _, target := range slices.Sorted(maps.Keys(targets)) {
		peer := connectionEndpoint(connectionSpec, target, true)
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
		if err := client.Delete(ctx, name, precondition.Unchanged(current)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("revoke unresolved connection policy %s: %w", name, err)
		}
		return nil
	}

	desired := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "cilium.io/v2", "kind": "CiliumNetworkPolicy", "specs": specs,
	}}
	desired.SetName(name)
	desired.SetNamespace(connectionSpec.GetK8SNamespace())
	desired.SetLabels(labelsDesired)
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
		return fmt.Errorf("publish connection policy %s: %w", name, err)
	}
	return nil
}

func connectionEndpoint(connectionSpec *ctrlv1.PrivateNetworkConnection, deployment string, target bool) map[string]interface{} {
	identity := labels.New().WorkspaceID(connectionSpec.GetWorkspaceId()).ProjectID(connectionSpec.GetProjectId()).DeploymentID(deployment)
	if target {
		identity.AppID(connectionSpec.GetTargetAppId())
	}
	return cilium.DeploymentEndpoint(identity)
}
