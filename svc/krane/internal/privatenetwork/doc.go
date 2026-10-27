// Package privatenetwork owns Krane's private network Services, Cilium
// policies, and source EndpointSlices. Shared headless Services publish one
// deployment port; per-caller ExternalName Services resolve connection aliases
// to them for CoreDNS.
package privatenetwork
