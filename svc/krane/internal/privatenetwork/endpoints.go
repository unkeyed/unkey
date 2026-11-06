package privatenetwork

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/netip"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"
)

const (
	sourceSliceManager = "private-dns.unkey.com"
	endpointsPerSlice  = 100
	intervalResync     = time.Minute
	delaySettle        = time.Second
)

var (
	podIdentityKeys = []string{
		labels.LabelKeyManagedBy, labels.LabelKeyWorkspaceID, labels.LabelKeyProjectID,
		labels.LabelKeyAppID, labels.LabelKeyDeploymentID,
	}
	deploymentPodLabels = labels.New().ComponentDeployment()
)

// MUST KEEP this publisher and selectorless Services until Cilium preserves
// readiness and termination in imported Kubernetes EndpointSlices. Its legacy
// ClusterMesh path exports terminating-but-serving Pods and imports them as
// ready, so Kubernetes-managed source slices alone would expose them to CoreDNS.
// Service-v2 preserves conditions internally but does not publish the remote
// Kubernetes slices CoreDNS reads (verified in 1.20.2 and 1.21.0-pre.3).
// Remove only after the upstream importer supports this and termination and
// final-endpoint withdrawal tests pass: https://github.com/cilium/cilium/issues/41533.
func (r *Reconciler) runEndpoints(ctx context.Context, changed <-chan struct{}) {
	for {
		if err := r.reconcileEndpoints(ctx); err != nil {
			logger.Warn("private network endpoint reconciliation failed", "error", err)
		}
		// The settle delay lets the caches observe this pass's own writes, so
		// the next pass does not overwrite them from a stale view.
		if !r.sleep(ctx, delaySettle, nil) || !r.sleep(ctx, intervalResync, changed) {
			return
		}
	}
}

func (r *Reconciler) reconcileEndpoints(ctx context.Context) error {
	started, completed := r.clock.Now(), false
	defer func() { r.observePass(loopEndpoints, started, completed) }()
	ctx, cancel := context.WithTimeout(ctx, timeoutPass)
	defer cancel()

	services, err := r.services.List(discoverySelector)
	if err != nil {
		return countError(loopEndpoints, stageList, fmt.Errorf("list private network Services for endpoint refresh: %w", err))
	}

	var errs []error
	for _, service := range services {
		if !sharedService(service) {
			continue
		}
		if err := r.ensureEndpoints(ctx, service); err != nil {
			errs = append(errs, countError(loopEndpoints, stageEndpointSlice, err))
		}
	}
	completed = true
	return errors.Join(errs...)
}

func (r *Reconciler) ensureEndpoints(ctx context.Context, service *corev1.Service) error {
	if service.DeletionTimestamp != nil || service.Spec.ClusterIP != corev1.ClusterIPNone ||
		len(service.Spec.Selector) != 0 || service.Spec.PublishNotReadyAddresses || len(service.Spec.Ports) != 1 ||
		service.Spec.Ports[0].Port < 1 || service.Spec.Ports[0].Port > 65535 {
		return fmt.Errorf("invalid source discovery Service %s/%s", service.Namespace, service.Name)
	}

	existing, err := r.endpointSlices.EndpointSlices(service.Namespace).List(k8slabels.SelectorFromSet(k8slabels.Set{
		discoveryv1.LabelManagedBy: sourceSliceManager, discoveryv1.LabelServiceName: service.Name,
	}))
	if err != nil {
		return fmt.Errorf("list source EndpointSlices: %w", err)
	}
	pods, err := r.deploymentPods(service.Namespace, service.Labels[labels.LabelKeyDeploymentID])
	if err != nil {
		return err
	}

	client := r.client.DiscoveryV1().EndpointSlices(service.Namespace)
	sliceLabels := maps.Clone(service.Labels)
	sliceLabels[discoveryv1.LabelServiceName] = service.Name
	sliceLabels[discoveryv1.LabelManagedBy] = sourceSliceManager

	byName := make(map[string]*discoveryv1.EndpointSlice, len(existing))
	for _, slice := range existing {
		owner := metav1.GetControllerOf(slice)
		if owner == nil || owner.APIVersion != "v1" || owner.Kind != "Service" || owner.Name != service.Name {
			return fmt.Errorf("refuse to replace foreign source EndpointSlice %s/%s", slice.Namespace, slice.Name)
		}
		if owner.UID != service.UID {
			if err := client.Delete(ctx, slice.Name, precondition.Unchanged(slice)); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("delete old Service incarnation EndpointSlice %s/%s: %w", slice.Namespace, slice.Name, err)
			}
			continue
		}
		byName[slice.Name] = slice
	}

	endpoints := readyEndpoints(service, pods, r.clusterKey.GetRegion())
	ownerRef := *metav1.NewControllerRef(service, corev1.SchemeGroupVersion.WithKind("Service"))
	for offset := 0; offset < max(1, len(endpoints)); offset += endpointsPerSlice {
		name := fmt.Sprintf("%s-%d", service.Name, offset/endpointsPerSlice)
		desired := &discoveryv1.EndpointSlice{
			TypeMeta: metav1.TypeMeta{APIVersion: discoveryv1.SchemeGroupVersion.String(), Kind: "EndpointSlice"},
			ObjectMeta: metav1.ObjectMeta{
				Name:            name,
				Namespace:       service.Namespace,
				Labels:          sliceLabels,
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
			AddressType: discoveryv1.AddressTypeIPv4,
			Endpoints:   endpoints[offset:min(offset+endpointsPerSlice, len(endpoints))],
			Ports: []discoveryv1.EndpointPort{{
				Name:        ptr.To("app"),
				Protocol:    ptr.To(corev1.ProtocolTCP),
				Port:        ptr.To(service.Spec.Ports[0].Port),
				AppProtocol: nil,
			}},
		}
		current, ok := byName[name]
		if !ok {
			if _, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager}); err != nil {
				return fmt.Errorf("create source EndpointSlice %s/%s: %w", service.Namespace, name, err)
			}
			continue
		}
		delete(byName, name)
		if maps.Equal(current.Labels, desired.Labels) && current.AddressType == desired.AddressType &&
			equality.Semantic.DeepEqual(current.Endpoints, desired.Endpoints) &&
			equality.Semantic.DeepEqual(current.Ports, desired.Ports) {
			continue
		}
		desired.ResourceVersion = current.ResourceVersion
		desired.UID = current.UID
		if _, err := client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager}); err != nil {
			return fmt.Errorf("update source EndpointSlice %s/%s: %w", service.Namespace, name, err)
		}
	}

	for _, slice := range byName {
		if err := client.Delete(ctx, slice.Name, precondition.Unchanged(slice)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete source EndpointSlice %s/%s: %w", slice.Namespace, slice.Name, err)
		}
	}
	return nil
}

func (r *Reconciler) deploymentPods(namespace, deploymentID string) ([]*corev1.Pod, error) {
	objects, err := r.pods.ByIndex(indexDeployment, namespace+"/"+deploymentID)
	if err != nil {
		return nil, fmt.Errorf("list local private network Pods: %w", err)
	}
	pods := make([]*corev1.Pod, 0, len(objects))
	for _, object := range objects {
		pod, ok := object.(*corev1.Pod)
		if !ok {
			return nil, fmt.Errorf("unexpected Pod cache entry %T", object)
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

func readyEndpoints(service *corev1.Service, pods []*corev1.Pod, region string) []discoveryv1.Endpoint {
	addresses := make(map[netip.Addr]struct{}, len(pods))
	for _, pod := range pods {
		if pod.UID == "" || pod.Spec.HostNetwork || pod.DeletionTimestamp != nil ||
			pod.Status.Phase != corev1.PodRunning || !deploymentPodLabels.Matches(pod.Labels) {
			continue
		}
		if slices.ContainsFunc(podIdentityKeys, func(key string) bool { return pod.Labels[key] != service.Labels[key] }) {
			continue
		}
		ready := slices.ContainsFunc(pod.Status.Conditions, func(c corev1.PodCondition) bool {
			return c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue
		})
		if !ready {
			continue
		}
		for _, ip := range pod.Status.PodIPs {
			if address, err := netip.ParseAddr(ip.IP); err == nil && privatenetwork.IsPodAddress(address) {
				addresses[address] = struct{}{}
			}
		}
	}

	var hostname *string
	if region != "" {
		hostname = ptr.To(region)
	}
	endpoints := make([]discoveryv1.Endpoint, 0, len(addresses))
	for _, address := range slices.SortedFunc(maps.Keys(addresses), netip.Addr.Compare) {
		endpoints = append(endpoints, discoveryv1.Endpoint{
			Addresses: []string{address.String()},
			Conditions: discoveryv1.EndpointConditions{
				Ready:       ptr.To(true),
				Serving:     ptr.To(true),
				Terminating: ptr.To(false),
			},
			Hostname:           hostname,
			TargetRef:          nil,
			DeprecatedTopology: nil,
			NodeName:           nil,
			Zone:               nil,
			Hints:              nil,
		})
	}
	return endpoints
}
