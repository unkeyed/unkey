package privatenetwork

import (
	"net/netip"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppendReadyAddresses appends private IPv4 addresses from ready, non-terminating
// endpoints in a slice controlled by the given Service. It does not deduplicate.
func AppendReadyAddresses(addresses []netip.Addr, service *corev1.Service, slice *discoveryv1.EndpointSlice) []netip.Addr {
	owner := metav1.GetControllerOf(slice)
	if slice.DeletionTimestamp != nil || owner == nil || owner.APIVersion != "v1" || owner.Kind != "Service" || owner.UID != service.UID || owner.Name != service.Name {
		return addresses
	}
	if slice.AddressType != discoveryv1.AddressTypeIPv4 {
		return addresses
	}

	for _, endpoint := range slice.Endpoints {
		if endpoint.Conditions.Ready == nil || !*endpoint.Conditions.Ready ||
			(endpoint.Conditions.Terminating != nil && *endpoint.Conditions.Terminating) {
			continue
		}
		for _, raw := range endpoint.Addresses {
			address, err := netip.ParseAddr(raw)
			if err == nil && address.Is4() && address.IsPrivate() {
				addresses = append(addresses, address)
			}
		}
	}
	return addresses
}
