package privatenetwork

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type absence struct {
	snapshotID string
	confirmed  bool
}

func (r *Reconciler) cleanupGrants(ctx context.Context, p *discoveryPass, snapshot *ctrlv1.PrivateNetworkStateChunk, connectionSpecs []*ctrlv1.PrivateNetworkConnection, confirm bool) error {
	wanted := make(map[string]struct{}, len(connectionSpecs))
	for _, connectionSpec := range connectionSpecs {
		wanted[publishedConnectionKey(connectionSpec)] = struct{}{}
	}
	if !snapshot.GetCertified() {
		maps.DeleteFunc(r.absences, func(key string, _ absence) bool {
			_, present := wanted[key]
			return present || !confirm
		})
		return nil
	}

	keys := make(map[string]struct{}, len(p.connections)+len(p.policies))
	for key := range p.connections {
		keys[key] = struct{}{}
	}
	for key, policy := range p.policies {
		if owned(policy.GetLabels()) {
			keys[key] = struct{}{}
		}
	}

	absences := make(map[string]absence)
	deferred := 0
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		if _, present := wanted[key]; present || !confirm {
			continue
		}
		previous, seen := r.absences[key]
		if !seen {
			previous = absence{snapshotID: snapshot.GetSnapshotId(), confirmed: false}
		} else if previous.snapshotID != snapshot.GetSnapshotId() {
			previous.confirmed = true
		}
		absences[key] = previous
		if !previous.confirmed {
			deferred++
			continue
		}

		if policy := p.policies[key]; policy != nil && owned(policy.GetLabels()) {
			if err := r.dynamic.Resource(cilium.NetworkPolicyResource).Namespace(policy.GetNamespace()).Delete(ctx, policy.GetName(), precondition.Unchanged(policy)); err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("delete connection policy %s: %w", key, err))
				continue
			}
			delete(p.policies, key)
		}
		remaining := slices.DeleteFunc(p.connections[key], func(connection *corev1.Service) bool {
			err := r.client.CoreV1().Services(connection.Namespace).Delete(ctx, connection.Name, precondition.Unchanged(connection))
			if err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("delete connection DNS %s/%s: %w", connection.Namespace, connection.Name, err))
				return false
			}
			delete(p.servicesByKey, connection.Namespace+"/"+connection.Name)
			return true
		})
		if len(remaining) == 0 {
			delete(p.connections, key)
		} else {
			p.connections[key] = remaining
		}
	}
	r.absences = absences
	metrics.PrivateNetworkDeferredDeletions.Set(float64(deferred))
	return errors.Join(errs...)
}
