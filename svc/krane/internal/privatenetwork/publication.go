package privatenetwork

import (
	"context"
	"fmt"
	"maps"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/appbinding"
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

func appLabels(app *ctrlv1.PrivateNetworkApp) labels.Labels {
	return labels.New().
		ManagedByKrane().
		WorkspaceID(app.GetWorkspaceId()).
		ProjectID(app.GetProjectId()).
		AppID(app.GetAppId())
}

func (r *Reconciler) ensureService(ctx context.Context, app *ctrlv1.PrivateNetworkApp, name string, existing *corev1.Service) (*corev1.Service, error) {
	client := r.client.CoreV1().Services(app.GetK8SNamespace())
	desiredLabels := appLabels(app).DeploymentID(app.GetDeploymentId())
	desiredLabels[labels.LabelKeyComponent] = component
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   app.GetK8SNamespace(),
			Labels:      desiredLabels,
			Annotations: map[string]string{ciliumGlobal: "true", ciliumGlobalSlices: "true"},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{{
				Name:       "app",
				Port:       app.GetPort(),
				TargetPort: intstr.FromInt32(app.GetPort()),
				Protocol:   corev1.ProtocolTCP,
			}},
			PublishNotReadyAddresses: false,
		},
	}

	if existing != nil {
		if !ownedByApp(existing.Labels, app) {
			return nil, fmt.Errorf("refuse to replace foreign Service %s/%s", app.GetK8SNamespace(), name)
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
			return nil, fmt.Errorf("update private network Service %s/%s: %w", app.GetK8SNamespace(), name, err)
		}
		return updated, nil
	}

	created, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	if err != nil {
		return nil, fmt.Errorf("create private network Service %s/%s: %w", app.GetK8SNamespace(), name, err)
	}
	return created, nil
}

func (r *Reconciler) ensureBinding(ctx context.Context, app *ctrlv1.PrivateNetworkApp, name string, service *corev1.Service, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	client := r.client.CoreV1().ConfigMaps(app.GetK8SNamespace())
	key := app.GetK8SNamespace() + "/" + name
	desiredData := map[string]string{
		"appSlug":      app.GetBindingName(),
		"deploymentId": app.GetDeploymentId(),
		"serviceName":  "",
	}
	if service != nil {
		desiredData["serviceName"] = service.Name
	}

	if existing != nil {
		if !owned(existing.Labels) || existing.Labels[labels.LabelKeyWorkspaceID] != app.GetWorkspaceId() ||
			existing.Labels[labels.LabelKeyProjectID] != app.GetProjectId() ||
			existing.Labels[labels.LabelKeyBindingID] != app.GetBindingId() ||
			existing.Labels[labels.LabelKeyCallerDeploymentID] != app.GetCallerDeploymentId() {
			return nil, fmt.Errorf("refuse to replace foreign ConfigMap %s", key)
		}
		revision, parseErr := strconv.ParseUint(existing.Data["revision"], 10, 64)
		if parseErr != nil || revision == 0 {
			return nil, fmt.Errorf("invalid existing binding revision for %s", key)
		}

		if existing.Data["appSlug"] == desiredData["appSlug"] && existing.Data["deploymentId"] == desiredData["deploymentId"] && existing.Data["serviceName"] == desiredData["serviceName"] && maps.Equal(existing.Labels, bindingLabels(app)) {
			return existing, nil
		}

		if service != nil && existing.Data["appSlug"] == desiredData["appSlug"] && maps.Equal(existing.Labels, bindingLabels(app)) {
			ready, err := r.hasReadyEndpoints(ctx, service)
			if err != nil {
				return nil, err
			}
			if !ready {
				return existing, nil
			}
		}

		if revision == ^uint64(0) {
			return nil, fmt.Errorf("binding revision exhausted for %s", key)
		}
		desiredData["revision"] = strconv.FormatUint(revision+1, 10)
		desired := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: app.GetK8SNamespace(), Labels: bindingLabels(app),
			ResourceVersion: existing.ResourceVersion, UID: existing.UID,
		}, Data: desiredData}
		updated, err := client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager})
		if err != nil {
			return nil, fmt.Errorf("update private network binding %s: %w", key, err)
		}
		logBindingPublished(app, key, existing.Data["deploymentId"], updated)
		return updated, nil
	}

	desiredData["revision"] = "1"
	desired := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: app.GetK8SNamespace(), Labels: bindingLabels(app),
	}, Data: desiredData}
	created, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	if err != nil {
		return nil, fmt.Errorf("create private network binding %s: %w", key, err)
	}
	logBindingPublished(app, key, "", created)
	return created, nil
}

func logBindingPublished(app *ctrlv1.PrivateNetworkApp, key, previousDeployment string, binding *corev1.ConfigMap) {
	logger.Info("private network binding published",
		"binding_key", key, "kind", entryKind(app), "workspace_id", app.GetWorkspaceId(),
		"binding_id", app.GetBindingId(), "caller_deployment_id", app.GetCallerDeploymentId(),
		"alias", app.GetBindingName(), "previous_deployment_id", previousDeployment,
		"deployment_id", binding.Data["deploymentId"], "revision", binding.Data["revision"])
}

func (r *Reconciler) hasReadyEndpoints(ctx context.Context, service *corev1.Service) (bool, error) {
	slices, err := r.client.DiscoveryV1().EndpointSlices(service.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: discoveryv1.LabelServiceName + "=" + service.Name,
	})
	if err != nil {
		return false, fmt.Errorf("check discovery readiness for %s/%s: %w", service.Namespace, service.Name, err)
	}
	for i := range slices.Items {
		if len(appbinding.AppendReadyAddresses(nil, service, &slices.Items[i])) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func bindingLabels(app *ctrlv1.PrivateNetworkApp) labels.Labels {
	l := appLabels(app)
	l[labels.LabelKeyComponent] = component
	l[labels.LabelKeyCallerDeploymentID] = app.GetCallerDeploymentId()
	l[labels.LabelKeyBindingID] = app.GetBindingId()
	return l
}
