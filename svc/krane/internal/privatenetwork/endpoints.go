package privatenetwork

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/netip"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	sourceSliceManager = "private-dns.unkey.com"
	endpointsPerSlice  = 100
)

func (r *Reconciler) runEndpoints(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if err := r.reconcileEndpoints(ctx); err != nil {
			logger.Warn("private network endpoint reconciliation failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Reconciler) reconcileEndpoints(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	r.endpointMu.Lock()
	defer r.endpointMu.Unlock()
	selector := labels.New().ManagedByKrane()
	selector[labels.LabelKeyComponent] = component
	services, err := r.client.CoreV1().Services("").List(ctx, metav1.ListOptions{LabelSelector: selector.ToString()})
	if err != nil {
		return fmt.Errorf("list private network Services for endpoint refresh: %w", err)
	}
	pods, err := r.localPods(ctx)
	if err != nil {
		return err
	}
	sourceSlices, err := r.sourceSlices(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for i := range services.Items {
		service := &services.Items[i]
		if !owned(service.Labels) || service.Labels[labels.LabelKeyDeploymentID] == "" ||
			len(service.Spec.Ports) != 1 || service.Name != discoveryName(service.Labels[labels.LabelKeyDeploymentID], service.Spec.Ports[0].Port) {
			continue
		}
		if err := r.ensureEndpoints(ctx, service, pods, sourceSlices[service.Namespace+"/"+service.Name]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *Reconciler) localPods(ctx context.Context) ([]corev1.Pod, error) {
	pods, err := r.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: labels.New().ManagedByKrane().ComponentDeployment().ToString(),
	})
	if err != nil {
		return nil, fmt.Errorf("list local private network Pods: %w", err)
	}
	return pods.Items, nil
}

func (r *Reconciler) sourceSlices(ctx context.Context) (map[string][]discoveryv1.EndpointSlice, error) {
	list, err := r.client.DiscoveryV1().EndpointSlices("").List(ctx, metav1.ListOptions{
		LabelSelector: discoveryv1.LabelManagedBy + "=" + sourceSliceManager,
	})
	if err != nil {
		return nil, fmt.Errorf("list source EndpointSlices: %w", err)
	}
	byService := make(map[string][]discoveryv1.EndpointSlice)
	for i := range list.Items {
		item := list.Items[i]
		key := item.Namespace + "/" + item.Labels[discoveryv1.LabelServiceName]
		byService[key] = append(byService[key], item)
	}
	return byService, nil
}

func (r *Reconciler) ensureEndpoints(ctx context.Context, service *corev1.Service, pods []corev1.Pod, existing []discoveryv1.EndpointSlice) error {
	if service.DeletionTimestamp != nil || service.Spec.ClusterIP != corev1.ClusterIPNone ||
		len(service.Spec.Selector) != 0 || service.Spec.PublishNotReadyAddresses || len(service.Spec.Ports) != 1 ||
		service.Spec.Ports[0].Port < 1 || service.Spec.Ports[0].Port > 65535 {
		return fmt.Errorf("invalid source discovery Service %s/%s", service.Namespace, service.Name)
	}
	client := r.client.DiscoveryV1().EndpointSlices(service.Namespace)
	sliceLabels := maps.Clone(service.Labels)
	sliceLabels[discoveryv1.LabelServiceName] = service.Name
	sliceLabels[discoveryv1.LabelManagedBy] = sourceSliceManager
	byName := make(map[string]*discoveryv1.EndpointSlice, len(existing))
	for i := range existing {
		item := &existing[i]
		owner := metav1.GetControllerOf(item)
		if !maps.Equal(item.Labels, sliceLabels) || owner == nil || owner.APIVersion != "v1" ||
			owner.Kind != "Service" || owner.Name != service.Name {
			return fmt.Errorf("refuse to replace foreign source EndpointSlice %s/%s", item.Namespace, item.Name)
		}
		if owner.UID != service.UID {
			if err := client.Delete(ctx, item.Name, deleteOptions(item)); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("delete old Service incarnation EndpointSlice %s/%s: %w", item.Namespace, item.Name, err)
			}
			continue
		}
		byName[item.Name] = item
	}

	endpoints := readyEndpoints(service, pods)
	for offset := 0; offset < max(1, len(endpoints)); offset += endpointsPerSlice {
		name := fmt.Sprintf("%s-%d", service.Name, offset/endpointsPerSlice)
		desired := &discoveryv1.EndpointSlice{
			TypeMeta: metav1.TypeMeta{APIVersion: discoveryv1.SchemeGroupVersion.String(), Kind: "EndpointSlice"},
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: service.Namespace, Labels: sliceLabels,
				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(service, corev1.SchemeGroupVersion.WithKind("Service"))},
			},
			AddressType: discoveryv1.AddressTypeIPv4,
			Endpoints:   endpoints[offset:min(offset+endpointsPerSlice, len(endpoints))],
			Ports: []discoveryv1.EndpointPort{{
				Name: ptr.To("app"), Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(service.Spec.Ports[0].Port),
				AppProtocol: nil,
			}},
		}
		if current, ok := byName[name]; ok {
			delete(byName, name)
			if equality.Semantic.DeepEqual(current.Endpoints, desired.Endpoints) &&
				equality.Semantic.DeepEqual(current.Ports, desired.Ports) && current.AddressType == desired.AddressType {
				continue
			}
			desired.ResourceVersion = current.ResourceVersion
			desired.UID = current.UID
			if _, err := client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager}); err != nil {
				return fmt.Errorf("update source EndpointSlice %s/%s: %w", service.Namespace, name, err)
			}
		} else if _, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager}); err != nil {
			return fmt.Errorf("create source EndpointSlice %s/%s: %w", service.Namespace, name, err)
		}
	}
	for _, item := range byName {
		if err := client.Delete(ctx, item.Name, deleteOptions(item)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete source EndpointSlice %s/%s: %w", item.Namespace, item.Name, err)
		}
	}
	return nil
}

func readyEndpoints(service *corev1.Service, pods []corev1.Pod) []discoveryv1.Endpoint {
	addresses := make(map[netip.Addr]struct{})
	for i := range pods {
		pod := &pods[i]
		if pod.Namespace != service.Namespace || pod.UID == "" || pod.Spec.HostNetwork ||
			pod.DeletionTimestamp != nil || pod.Status.Phase != corev1.PodRunning ||
			pod.Labels[labels.LabelKeyComponent] != "deployment" {
			continue
		}
		matches := true
		for _, key := range []string{
			labels.LabelKeyManagedBy, labels.LabelKeyWorkspaceID, labels.LabelKeyProjectID,
			labels.LabelKeyAppID, labels.LabelKeyDeploymentID,
		} {
			if pod.Labels[key] != service.Labels[key] {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		ready := slices.ContainsFunc(pod.Status.Conditions, func(c corev1.PodCondition) bool {
			return c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue
		})
		if !ready {
			continue
		}
		for _, ip := range pod.Status.PodIPs {
			address, err := netip.ParseAddr(ip.IP)
			if err == nil && address.Is4() && address.IsPrivate() {
				addresses[address] = struct{}{}
			}
		}
	}
	ordered := slices.SortedFunc(maps.Keys(addresses), func(a, b netip.Addr) int { return a.Compare(b) })
	endpoints := make([]discoveryv1.Endpoint, 0, len(ordered))
	for _, address := range ordered {
		endpoints = append(endpoints, discoveryv1.Endpoint{
			Addresses: []string{address.String()},
			Conditions: discoveryv1.EndpointConditions{
				Ready: ptr.To(true), Serving: ptr.To(true), Terminating: ptr.To(false),
			},
			Hostname: nil, TargetRef: nil, DeprecatedTopology: nil, NodeName: nil, Zone: nil, Hints: nil,
		})
	}
	return endpoints
}
