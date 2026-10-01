package privatenetwork

import (
	"context"
	"fmt"
	"maps"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/appconnection"
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
	desiredLabels := targetAppLabels(connectionSpec).DeploymentID(connectionSpec.GetTargetDeploymentId())
	desiredLabels[labels.LabelKeyComponent] = component
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   connectionSpec.GetK8SNamespace(),
			Labels:      desiredLabels,
			Annotations: map[string]string{ciliumGlobal: "true", ciliumGlobalSlices: "true"},
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

func (r *Reconciler) ensureConnection(ctx context.Context, connectionSpec *ctrlv1.PrivateNetworkConnection, name string, service *corev1.Service, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	client := r.client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace())
	key := connectionSpec.GetK8SNamespace() + "/" + name
	desiredData := map[string]string{
		"appSlug":      connectionSpec.GetConnectionName(),
		"deploymentId": connectionSpec.GetTargetDeploymentId(),
		"serviceName":  "",
	}
	if service != nil {
		desiredData["serviceName"] = service.Name
	}

	if existing != nil {
		if !owned(existing.Labels) || existing.Labels[labels.LabelKeyWorkspaceID] != connectionSpec.GetWorkspaceId() ||
			existing.Labels[labels.LabelKeyProjectID] != connectionSpec.GetProjectId() ||
			existing.Labels[labels.LabelKeyConnectionID] != connectionSpec.GetConnectionId() ||
			existing.Labels[labels.LabelKeyCallerDeploymentID] != connectionSpec.GetCallerDeploymentId() {
			return nil, fmt.Errorf("refuse to replace foreign ConfigMap %s", key)
		}
		revision, parseErr := strconv.ParseUint(existing.Data["revision"], 10, 64)
		if parseErr != nil || revision == 0 {
			return nil, fmt.Errorf("invalid existing connection revision for %s", key)
		}

		if existing.Data["appSlug"] == desiredData["appSlug"] && existing.Data["deploymentId"] == desiredData["deploymentId"] && existing.Data["serviceName"] == desiredData["serviceName"] && maps.Equal(existing.Labels, connectionLabels(connectionSpec)) {
			return existing, nil
		}

		if service != nil && existing.Data["appSlug"] == desiredData["appSlug"] && maps.Equal(existing.Labels, connectionLabels(connectionSpec)) {
			ready, err := r.hasReadyEndpoints(ctx, service)
			if err != nil {
				return nil, err
			}
			if !ready {
				return existing, nil
			}
		}

		if revision == ^uint64(0) {
			return nil, fmt.Errorf("connection revision exhausted for %s", key)
		}
		desiredData["revision"] = strconv.FormatUint(revision+1, 10)
		desired := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: connectionSpec.GetK8SNamespace(), Labels: connectionLabels(connectionSpec),
			ResourceVersion: existing.ResourceVersion, UID: existing.UID,
		}, Data: desiredData}
		updated, err := client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager})
		if err != nil {
			return nil, fmt.Errorf("update private network connection %s: %w", key, err)
		}
		logConnectionPublished(connectionSpec, key, existing.Data["deploymentId"], updated)
		return updated, nil
	}

	desiredData["revision"] = "1"
	desired := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: connectionSpec.GetK8SNamespace(), Labels: connectionLabels(connectionSpec),
	}, Data: desiredData}
	created, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
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
		"deployment_id", connection.Data["deploymentId"], "revision", connection.Data["revision"])
}

func (r *Reconciler) hasReadyEndpoints(ctx context.Context, service *corev1.Service) (bool, error) {
	slices, err := r.client.DiscoveryV1().EndpointSlices(service.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: discoveryv1.LabelServiceName + "=" + service.Name,
	})
	if err != nil {
		return false, fmt.Errorf("check discovery readiness for %s/%s: %w", service.Namespace, service.Name, err)
	}
	for i := range slices.Items {
		if len(appconnection.AppendReadyAddresses(nil, service, &slices.Items[i])) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func connectionLabels(connectionSpec *ctrlv1.PrivateNetworkConnection) labels.Labels {
	l := targetAppLabels(connectionSpec)
	l[labels.LabelKeyComponent] = component
	l[labels.LabelKeyCallerDeploymentID] = connectionSpec.GetCallerDeploymentId()
	l[labels.LabelKeyConnectionID] = connectionSpec.GetConnectionId()
	return l
}
