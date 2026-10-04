package privatenetwork

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func connectionResourceName(connectionSpec *ctrlv1.PrivateNetworkConnection) string {
	return resourceName("unkey-pn-connection", connectionSpec.GetConnectionId()+"/"+connectionSpec.GetCallerDeploymentId())
}

func (r *Reconciler) cleanupServices(ctx context.Context, services *corev1.ServiceList, desiredServices map[string]struct{}) error {
	for i := range services.Items {
		item := &services.Items[i]
		if _, ok := desiredServices[item.Namespace+"/"+item.Name]; ok || !owned(item.Labels) {
			continue
		}
		deadline, ok := retirementDeadline(item)
		if !ok {
			updated := item.DeepCopy()
			updated.Annotations = maps.Clone(item.Annotations)
			if updated.Annotations == nil {
				updated.Annotations = make(map[string]string)
			}
			updated.Annotations[privatenetwork.RetireAfterAnnotation] = r.clock.Now().Add(privatenetwork.ReplacementOverlap).UTC().Format(time.RFC3339Nano)
			if _, err := r.client.CoreV1().Services(item.Namespace).Update(ctx, updated, metav1.UpdateOptions{FieldManager: fieldManager}); err != nil {
				return fmt.Errorf("schedule obsolete private network Service %s/%s retirement: %w", item.Namespace, item.Name, err)
			}
			continue
		}
		if r.clock.Now().Before(deadline) {
			continue
		}
		if err := r.client.CoreV1().Services(item.Namespace).Delete(ctx, item.Name, precondition.Unchanged(item)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete private network Service %s/%s: %w", item.Namespace, item.Name, err)
		}
	}
	return nil
}

func retirementDeadline(service *corev1.Service) (time.Time, bool) {
	raw := service.Annotations[privatenetwork.RetireAfterAnnotation]
	if raw == "" {
		return time.Time{}, false
	}
	deadline, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, true
	}
	return deadline, true
}

func owned(l map[string]string) bool {
	return labels.New().ManagedByKrane().Component(component).Matches(l) &&
		l[labels.LabelKeyWorkspaceID] != "" &&
		l[labels.LabelKeyProjectID] != "" &&
		l[labels.LabelKeyAppID] != ""
}

func ownedByTargetApp(l map[string]string, connectionSpec *ctrlv1.PrivateNetworkConnection) bool {
	return owned(l) &&
		l[labels.LabelKeyWorkspaceID] == connectionSpec.GetWorkspaceId() &&
		l[labels.LabelKeyProjectID] == connectionSpec.GetProjectId() &&
		l[labels.LabelKeyAppID] == connectionSpec.GetTargetAppId()
}

func discoveryName(deployment string, port int32) string {
	return privatenetwork.ServiceName(deployment, port)
}

func resourceName(prefix, id string) string {
	sum := sha256.Sum256([]byte(id))
	return prefix + "-" + hex.EncodeToString(sum[:16])
}
