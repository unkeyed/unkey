package privatenetwork

import (
	"context"
	"fmt"
	"maps"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
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
		if !ownedByTargetApp(existing.Labels, connectionSpec) {
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

func (r *Reconciler) ensureConnection(ctx context.Context, connectionSpec *ctrlv1.PrivateNetworkConnection, name string, service *corev1.Service, existing *corev1.ConfigMap, endpointSlices []discoveryv1.EndpointSlice) (*corev1.ConfigMap, error) {
	client := r.client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace())
	key := connectionSpec.GetK8SNamespace() + "/" + name
	desired := privatenetwork.ConnectionData{
		Alias:        connectionSpec.GetConnectionName(),
		DeploymentID: connectionSpec.GetTargetDeploymentId(),
		ServiceName:  "",
		Revision:     1,
	}
	if service != nil {
		desired.ServiceName = service.Name
	}

	if existing != nil {
		if !owned(existing.Labels) || existing.Labels[labels.LabelKeyWorkspaceID] != connectionSpec.GetWorkspaceId() ||
			existing.Labels[labels.LabelKeyProjectID] != connectionSpec.GetProjectId() ||
			existing.Labels[labels.LabelKeyConnectionID] != connectionSpec.GetConnectionId() ||
			existing.Labels[labels.LabelKeyCallerDeploymentID] != connectionSpec.GetCallerDeploymentId() {
			return nil, fmt.Errorf("refuse to replace foreign ConfigMap %s", key)
		}
		current, decodeErr := privatenetwork.Decode(existing.Data)
		if decodeErr != nil {
			return nil, fmt.Errorf("invalid existing connection data for %s: %w", key, decodeErr)
		}

		desired.Revision = current.Revision
		if current == desired && maps.Equal(existing.Labels, connectionLabels(connectionSpec)) {
			return existing, nil
		}

		if service != nil && current.Alias == desired.Alias && maps.Equal(existing.Labels, connectionLabels(connectionSpec)) {
			if !hasReadyEndpoints(service, endpointSlices) {
				listed, listErr := r.client.DiscoveryV1().EndpointSlices(service.Namespace).List(ctx, metav1.ListOptions{
					LabelSelector: discoveryv1.LabelServiceName + "=" + service.Name,
				})
				if listErr != nil {
					return nil, fmt.Errorf("check discovery readiness for %s/%s: %w", service.Namespace, service.Name, listErr)
				}
				if !hasReadyEndpoints(service, listed.Items) {
					return existing, nil
				}
			}
		}

		if current.Revision == ^uint64(0) {
			return nil, fmt.Errorf("connection revision exhausted for %s", key)
		}
		desired.Revision++
		desiredData, encodeErr := privatenetwork.Encode(desired)
		if encodeErr != nil {
			return nil, fmt.Errorf("encode private network connection %s: %w", key, encodeErr)
		}

		desiredConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name,
				Namespace:       connectionSpec.GetK8SNamespace(),
				Labels:          connectionLabels(connectionSpec),
				ResourceVersion: existing.ResourceVersion,
				UID:             existing.UID,
			},
			Data: desiredData,
		}
		updated, err := client.Update(ctx, desiredConfigMap, metav1.UpdateOptions{FieldManager: fieldManager})
		if err != nil {
			return nil, fmt.Errorf("update private network connection %s: %w", key, err)
		}
		logConnectionPublished(connectionSpec, key, current.DeploymentID, updated)
		return updated, nil
	}

	desiredData, err := privatenetwork.Encode(desired)
	if err != nil {
		return nil, fmt.Errorf("encode private network connection %s: %w", key, err)
	}

	desiredConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: connectionSpec.GetK8SNamespace(),
			Labels:    connectionLabels(connectionSpec),
		},
		Data: desiredData,
	}
	created, err := client.Create(ctx, desiredConfigMap, metav1.CreateOptions{FieldManager: fieldManager})
	if err != nil {
		return nil, fmt.Errorf("create private network connection %s: %w", key, err)
	}
	logConnectionPublished(connectionSpec, key, "", created)
	return created, nil
}

func logConnectionPublished(connectionSpec *ctrlv1.PrivateNetworkConnection, key, previousDeployment string, connection *corev1.ConfigMap) {
	logger.Info("private network connection published",
		"connection_key", key, "kind", entryKind(connectionSpec), "workspace_id", connectionSpec.GetWorkspaceId(),
		"connection_id", connectionSpec.GetConnectionId(), "caller_deployment_id", connectionSpec.GetCallerDeploymentId(),
		"alias", connectionSpec.GetConnectionName(), "previous_deployment_id", previousDeployment,
		"deployment_id", connection.Data[privatenetwork.ConnectionDeploymentKey], "revision", connection.Data[privatenetwork.ConnectionRevisionKey])
}

func hasReadyEndpoints(service *corev1.Service, slices []discoveryv1.EndpointSlice) bool {
	for i := range slices {
		if len(privatenetwork.AppendReadyAddresses(nil, service, &slices[i])) > 0 {
			return true
		}
	}
	return false
}

func connectionLabels(connectionSpec *ctrlv1.PrivateNetworkConnection) labels.Labels {
	l := targetAppLabels(connectionSpec).Component(component)
	l[labels.LabelKeyCallerDeploymentID] = connectionSpec.GetCallerDeploymentId()
	l[labels.LabelKeyConnectionID] = connectionSpec.GetConnectionId()
	return l
}
