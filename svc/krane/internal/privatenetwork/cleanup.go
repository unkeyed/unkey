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

func (r *Reconciler) cleanupGrants(ctx context.Context, snapshotID string, wanted map[string]struct{}, connections map[string][]*corev1.Service, policies map[string]*unstructured.Unstructured, confirm bool) error {
	keys := make(map[string]struct{}, len(connections)+len(policies))
	for key := range connections {
		keys[key] = struct{}{}
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

		policy := policies[key]
		if policy != nil && owned(policy.GetLabels()) {
			if err := r.dynamic.Resource(cilium.NetworkPolicyResource).Namespace(policy.GetNamespace()).Delete(ctx, policy.GetName(), precondition.Unchanged(policy)); err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("delete connection policy %s: %w", key, err))
				continue
			}
			delete(policies, key)
		}
		remaining := slices.DeleteFunc(connections[key], func(connection *corev1.Service) bool {
			err := r.client.CoreV1().Services(connection.Namespace).Delete(ctx, connection.Name, precondition.Unchanged(connection))
			if err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("delete connection DNS %s/%s: %w", connection.Namespace, connection.Name, err))
				return false
			}
			return true
		})
		if len(remaining) == 0 {
			delete(connections, key)
		} else {
			connections[key] = remaining
		}
	}
	r.absences = next
	metrics.PrivateNetworkDeferredDeletions.Set(float64(deferred))
	return errors.Join(errs...)
}
