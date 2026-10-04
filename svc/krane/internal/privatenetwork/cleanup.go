package privatenetwork

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type absence struct {
	snapshotID string
	confirmed  bool
}

func (r *Reconciler) cleanupGrants(ctx context.Context, snapshotID string, wanted map[string]struct{}, connections map[string]*corev1.ConfigMap, policies map[string]*unstructured.Unstructured, confirm bool) error {
	keys := make(map[string]struct{}, len(connections)+len(policies))
	for key, connection := range connections {
		if owned(connection.Labels) {
			keys[key] = struct{}{}
		}
	}
	for key, policy := range policies {
		if owned(policy.GetLabels()) {
			keys[key] = struct{}{}
		}
	}

	next := make(map[string]absence)
	deferred := 0
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		_, present := wanted[key]
		if present || !confirm {
			continue
		}
		previous, seen := r.absences[key]
		if !seen {
			previous = absence{snapshotID: snapshotID, confirmed: false}
		} else if previous.snapshotID != snapshotID {
			previous.confirmed = true
		}
		next[key] = previous
		if !previous.confirmed {
			deferred++
			continue
		}

		policy, connection := policies[key], connections[key]
		if policy != nil && owned(policy.GetLabels()) {
			if err := r.dynamic.Resource(cilium.NetworkPolicyResource).Namespace(policy.GetNamespace()).Delete(ctx, policy.GetName(), precondition.Unchanged(policy)); err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("delete connection policy %s: %w", key, err))
				continue
			}
			delete(policies, key)
		}
		if connection != nil && owned(connection.Labels) {
			if err := r.client.CoreV1().ConfigMaps(connection.Namespace).Delete(ctx, connection.Name, precondition.Unchanged(connection)); err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("delete connection DNS %s: %w", key, err))
				continue
			}
			delete(connections, key)
		}
	}
	r.absences = next
	metrics.PrivateNetworkDeferredDeletions.Set(float64(deferred))
	return errors.Join(errs...)
}
