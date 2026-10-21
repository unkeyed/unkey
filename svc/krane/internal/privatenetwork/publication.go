package privatenetwork

import (
	"context"
	"fmt"
	"maps"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
)

func publishedDeployment(connection *corev1.Service, services map[string]*corev1.Service) (string, error) {
	key := connection.Namespace + "/" + connection.Name
	deployment := connection.Annotations[privatenetwork.TargetDeploymentAnnotation]
	name := targetServiceName(connection)
	if connection.Spec.Type != corev1.ServiceTypeExternalName || deployment == "" || name == "" ||
		connection.Spec.ExternalName != externalName(connection.Annotations[privatenetwork.RegionAnnotation], name, connection.Namespace) {
		return "", fmt.Errorf("invalid published connection Service %s", key)
	}
	target := services[connection.Namespace+"/"+name]
	if target == nil {
		return deployment, nil
	}
	if !sharedService(target) || target.Labels[labels.LabelKeyDeploymentID] != deployment ||
		target.Labels[labels.LabelKeyWorkspaceID] != connection.Labels[labels.LabelKeyWorkspaceID] ||
		target.Labels[labels.LabelKeyProjectID] != connection.Labels[labels.LabelKeyProjectID] ||
		target.Labels[labels.LabelKeyAppID] != connection.Labels[labels.LabelKeyAppID] {
		return "", fmt.Errorf("published connection Service %s does not match target Service %s", key, target.Name)
	}
	return deployment, nil
}

func targetServiceName(connection *corev1.Service) string {
	host := connection.Spec.ExternalName
	if region := connection.Annotations[privatenetwork.RegionAnnotation]; region != "" {
		var found bool
		if host, found = strings.CutPrefix(host, region+"."); !found {
			return ""
		}
	}
	name, _, _ := strings.Cut(host, ".")
	return name
}

func externalName(region, target, namespace string) string {
	if region == "" {
		return privatenetwork.ServiceHost(target, namespace)
	}
	return region + "." + privatenetwork.ServiceHost(target, namespace)
}

func (r *Reconciler) ensureConnection(ctx context.Context, connectionSpec *ctrlv1.PrivateNetworkConnection, name, region string, target, existing *corev1.Service, previous string) (*corev1.Service, error) {
	namespace := connectionSpec.GetK8SNamespace()
	key := namespace + "/" + name
	client := r.client.CoreV1().Services(namespace)

	if existing != nil && !ownedByConnection(existing.Labels, connectionSpec) {
		return nil, fmt.Errorf("refuse to replace foreign Service %s", key)
	}
	if target == nil {
		if existing != nil {
			if err := client.Delete(ctx, existing.Name, precondition.Unchanged(existing)); err != nil && !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("withdraw unresolved private network connection %s: %w", key, err)
			}
		}
		return nil, nil
	}

	deployment := target.Labels[labels.LabelKeyDeploymentID]
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      connectionLabels(connectionSpec),
			Annotations: map[string]string{privatenetwork.TargetDeploymentAnnotation: deployment},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: externalName(region, target.Name, namespace),
		},
	}
	if region != "" {
		desired.Annotations[privatenetwork.RegionAnnotation] = region
	}

	if existing == nil {
		created, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
		if err != nil {
			return nil, fmt.Errorf("create private network connection %s: %w", key, err)
		}
		logConnectionPublished(connectionSpec, key, region, "", deployment)
		return created, nil
	}

	if maps.Equal(existing.Labels, desired.Labels) && maps.Equal(existing.Annotations, desired.Annotations) &&
		existing.Spec.Type == desired.Spec.Type && existing.Spec.ExternalName == desired.Spec.ExternalName && len(existing.Spec.Ports) == 0 {
		return existing, nil
	}
	if previous != "" && previous != deployment && maps.Equal(existing.Labels, desired.Labels) {
		ready, err := r.targetReady(ctx, target)
		if err != nil {
			return nil, err
		}
		if !ready {
			return existing, nil
		}
	}

	desired.ResourceVersion = existing.ResourceVersion
	desired.UID = existing.UID
	updated, err := client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager})
	if err != nil {
		return nil, fmt.Errorf("update private network connection %s: %w", key, err)
	}
	logConnectionPublished(connectionSpec, key, region, previous, deployment)
	return updated, nil
}

// targetReady falls back to the API server because slices imported from other
// clusters are not guaranteed to carry the labels that select the cache.
func (r *Reconciler) targetReady(ctx context.Context, target *corev1.Service) (bool, error) {
	selector := k8slabels.SelectorFromSet(k8slabels.Set{discoveryv1.LabelServiceName: target.Name})
	cached, err := r.endpointSlices.EndpointSlices(target.Namespace).List(selector)
	if err != nil {
		return false, fmt.Errorf("check cached discovery readiness for %s/%s: %w", target.Namespace, target.Name, err)
	}
	for _, slice := range cached {
		if len(privatenetwork.AppendReadyAddresses(nil, target, slice)) > 0 {
			return true, nil
		}
	}
	listed, err := r.client.DiscoveryV1().EndpointSlices(target.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return false, fmt.Errorf("check discovery readiness for %s/%s: %w", target.Namespace, target.Name, err)
	}
	for i := range listed.Items {
		if len(privatenetwork.AppendReadyAddresses(nil, target, &listed.Items[i])) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func logConnectionPublished(connectionSpec *ctrlv1.PrivateNetworkConnection, key, region, deploymentPrevious, deployment string) {
	logger.Info("private network connection published",
		"connection_key", key, "kind", entryKind(connectionSpec), "workspace_id", connectionSpec.GetWorkspaceId(),
		"connection_id", connectionSpec.GetConnectionId(), "caller_deployment_id", connectionSpec.GetCallerDeploymentId(),
		"alias", connectionSpec.GetConnectionName(), "region", region, "previous_deployment_id", deploymentPrevious,
		"deployment_id", deployment)
}

func connectionLabels(connectionSpec *ctrlv1.PrivateNetworkConnection) labels.Labels {
	l := targetAppLabels(connectionSpec).Component(component)
	l[labels.LabelKeyCallerDeploymentID] = connectionSpec.GetCallerDeploymentId()
	l[labels.LabelKeyConnectionID] = connectionSpec.GetConnectionId()
	return l
}
