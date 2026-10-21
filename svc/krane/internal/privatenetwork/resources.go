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
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	discoveryLabels   = labels.New().ManagedByKrane().Component(component)
	discoverySelector = k8slabels.SelectorFromSet(k8slabels.Set(discoveryLabels))
)

const (
	annotationCiliumGlobal       = "service.cilium.io/global"
	annotationCiliumGlobalSlices = "service.cilium.io/global-sync-endpoint-slices"
)

func targetAppLabels(connectionSpec *ctrlv1.PrivateNetworkConnection) labels.Labels {
	return labels.New().
		ManagedByKrane().
		WorkspaceID(connectionSpec.GetWorkspaceId()).
		ProjectID(connectionSpec.GetProjectId()).
		AppID(connectionSpec.GetTargetAppId())
}

func (r *Reconciler) ensureService(ctx context.Context, connectionSpec *ctrlv1.PrivateNetworkConnection, existing *corev1.Service) (*corev1.Service, error) {
	namespace := connectionSpec.GetK8SNamespace()
	name := privatenetwork.ServiceName(connectionSpec.GetTargetDeploymentId(), connectionSpec.GetTargetPort())
	client := r.client.CoreV1().Services(namespace)
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      targetAppLabels(connectionSpec).DeploymentID(connectionSpec.GetTargetDeploymentId()).Component(component),
			Annotations: map[string]string{annotationCiliumGlobal: "true", annotationCiliumGlobalSlices: "true"},
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
			return nil, fmt.Errorf("refuse to replace foreign Service %s/%s", namespace, name)
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
			return nil, fmt.Errorf("update private network Service %s/%s: %w", namespace, name, err)
		}
		return updated, nil
	}

	created, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	if apierrors.IsNotFound(err) {
		_, err = r.client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("create private network namespace %s: %w", namespace, err)
		}
		created, err = client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	}
	if err != nil {
		return nil, fmt.Errorf("create private network Service %s/%s: %w", namespace, name, err)
	}
	return created, nil
}

func grantName(connectionID, callerDeploymentID string) string {
	sum := sha256.Sum256([]byte(connectionID + "/" + callerDeploymentID))
	return "unkey-pn-connection-" + hex.EncodeToString(sum[:16])
}

func grantKey(connection *corev1.Service) string {
	return connection.Namespace + "/" + grantName(connection.Labels[labels.LabelKeyConnectionID], connection.Labels[labels.LabelKeyCallerDeploymentID])
}

func (r *Reconciler) cleanupServices(ctx context.Context, services []*corev1.Service, servicesDesired map[string]struct{}) error {
	now := r.clock.Now()
	for _, service := range services {
		if _, ok := servicesDesired[service.Namespace+"/"+service.Name]; ok || !sharedService(service) {
			continue
		}
		deadline, ok := retirementDeadline(service)
		if !ok {
			updated := service.DeepCopy()
			if updated.Annotations == nil {
				updated.Annotations = make(map[string]string, 1)
			}
			updated.Annotations[privatenetwork.RetireAfterAnnotation] = now.Add(privatenetwork.ReplacementOverlap).UTC().Format(time.RFC3339Nano)
			if _, err := r.client.CoreV1().Services(service.Namespace).Update(ctx, updated, metav1.UpdateOptions{FieldManager: fieldManager}); err != nil {
				return fmt.Errorf("schedule obsolete private network Service %s/%s retirement: %w", service.Namespace, service.Name, err)
			}
			continue
		}
		if now.Before(deadline) {
			continue
		}
		if err := r.client.CoreV1().Services(service.Namespace).Delete(ctx, service.Name, precondition.Unchanged(service)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete private network Service %s/%s: %w", service.Namespace, service.Name, err)
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
	return discoveryLabels.Matches(l) &&
		l[labels.LabelKeyWorkspaceID] != "" &&
		l[labels.LabelKeyProjectID] != "" &&
		l[labels.LabelKeyAppID] != ""
}

func sharedService(service *corev1.Service) bool {
	deployment := service.Labels[labels.LabelKeyDeploymentID]
	return owned(service.Labels) && !connectionService(service) && deployment != "" &&
		len(service.Spec.Ports) == 1 && service.Name == privatenetwork.ServiceName(deployment, service.Spec.Ports[0].Port)
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
