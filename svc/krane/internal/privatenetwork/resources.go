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
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) namespaces(ctx context.Context) (map[string]struct{}, error) {
	list, err := r.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	namespaces := make(map[string]struct{}, len(list.Items))
	for i := range list.Items {
		namespaces[list.Items[i].Name] = struct{}{}
	}
	return namespaces, nil
}

func targetAppLabels(connectionSpec *ctrlv1.PrivateNetworkConnection) labels.Labels {
	return labels.New().
		ManagedByKrane().
		WorkspaceID(connectionSpec.GetWorkspaceId()).
		ProjectID(connectionSpec.GetProjectId()).
		AppID(connectionSpec.GetTargetAppId())
}

func (r *Reconciler) ensureService(ctx context.Context, connectionSpec *ctrlv1.PrivateNetworkConnection, name string, existing *corev1.Service) (*corev1.Service, error) {
	client := r.client.CoreV1().Services(connectionSpec.GetK8SNamespace())
	desiredLabels := targetAppLabels(connectionSpec).DeploymentID(connectionSpec.GetTargetDeploymentId()).Component(component)
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   connectionSpec.GetK8SNamespace(),
			Labels:      desiredLabels,
			Annotations: map[string]string{privatenetwork.CiliumGlobalAnnotation: "true", privatenetwork.CiliumGlobalSlicesAnnotation: "true"},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{{
				Name:       "app",
				Port:       connectionSpec.GetTargetPort(),
				TargetPort: intstr.FromInt32(connectionSpec.GetTargetPort()),
				Protocol:   corev1.ProtocolTCP,
			}},
			PublishNotReadyAddresses: false,
		},
	}

	if existing != nil {
		if !ownedByTargetApp(existing.Labels, connectionSpec) || !sharedService(existing) {
			return nil, fmt.Errorf("refuse to replace foreign Service %s/%s", connectionSpec.GetK8SNamespace(), name)
		}

		if maps.Equal(existing.Labels, desired.Labels) && maps.Equal(existing.Annotations, desired.Annotations) &&
			maps.Equal(existing.Spec.Selector, desired.Spec.Selector) && existing.Spec.ClusterIP == corev1.ClusterIPNone &&
			!existing.Spec.PublishNotReadyAddresses && len(existing.Spec.Ports) == 1 &&
			existing.Spec.Ports[0] == desired.Spec.Ports[0] {
			return existing, nil
		}

		desired.ResourceVersion = existing.ResourceVersion
		desired.UID = existing.UID
		desired.Spec.ClusterIPs = existing.Spec.ClusterIPs
		desired.Spec.IPFamilies = existing.Spec.IPFamilies
		desired.Spec.IPFamilyPolicy = existing.Spec.IPFamilyPolicy
		updated, err := client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager})
		if err != nil {
			return nil, fmt.Errorf("update private network Service %s/%s: %w", connectionSpec.GetK8SNamespace(), name, err)
		}
		return updated, nil
	}

	created, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	if err != nil {
		return nil, fmt.Errorf("create private network Service %s/%s: %w", connectionSpec.GetK8SNamespace(), name, err)
	}
	return created, nil
}

func connectionResourceName(connectionSpec *ctrlv1.PrivateNetworkConnection) string {
	return grantName(connectionSpec.GetConnectionId(), connectionSpec.GetCallerDeploymentId())
}

func grantName(connectionID, callerDeploymentID string) string {
	return resourceName("unkey-pn-connection", connectionID+"/"+callerDeploymentID)
}

// grantKey identifies the connection a connection Service publishes. It
// matches the key of that connection's policy, so cleanup removes both.
func grantKey(connection *corev1.Service) string {
	return connection.Namespace + "/" + grantName(connection.Labels[labels.LabelKeyConnectionID], connection.Labels[labels.LabelKeyCallerDeploymentID])
}

func (r *Reconciler) cleanupServices(ctx context.Context, services []*corev1.Service, desiredServices map[string]struct{}) error {
	for _, item := range services {
		if _, ok := desiredServices[item.Namespace+"/"+item.Name]; ok || !sharedService(item) {
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

// sharedService reports whether service is a deployment port's shared
// headless Service rather than a connection Service.
func sharedService(service *corev1.Service) bool {
	deployment := service.Labels[labels.LabelKeyDeploymentID]
	return owned(service.Labels) && !connectionService(service) && deployment != "" &&
		len(service.Spec.Ports) == 1 && service.Name == discoveryName(deployment, service.Spec.Ports[0].Port)
}

func connectionService(service *corev1.Service) bool {
	return owned(service.Labels) && service.Labels[labels.LabelKeyConnectionID] != "" && service.Labels[labels.LabelKeyCallerDeploymentID] != ""
}

func ownedByTargetApp(l map[string]string, connectionSpec *ctrlv1.PrivateNetworkConnection) bool {
	return owned(l) &&
		l[labels.LabelKeyWorkspaceID] == connectionSpec.GetWorkspaceId() &&
		l[labels.LabelKeyProjectID] == connectionSpec.GetProjectId() &&
		l[labels.LabelKeyAppID] == connectionSpec.GetTargetAppId()
}

func ownedByConnection(l map[string]string, connectionSpec *ctrlv1.PrivateNetworkConnection) bool {
	return owned(l) &&
		l[labels.LabelKeyWorkspaceID] == connectionSpec.GetWorkspaceId() &&
		l[labels.LabelKeyProjectID] == connectionSpec.GetProjectId() &&
		l[labels.LabelKeyConnectionID] == connectionSpec.GetConnectionId() &&
		l[labels.LabelKeyCallerDeploymentID] == connectionSpec.GetCallerDeploymentId()
}

func discoveryName(deployment string, port int32) string {
	return privatenetwork.ServiceName(deployment, port)
}

func resourceName(prefix, id string) string {
	sum := sha256.Sum256([]byte(id))
	return prefix + "-" + hex.EncodeToString(sum[:16])
}
