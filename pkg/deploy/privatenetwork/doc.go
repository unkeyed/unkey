// Package privatenetwork defines the Kubernetes publication contract shared by
// Ctrl, Krane, and CoreDNS. It covers app connections and a deployment's own
// replicas, which are discoverable without a user-created connection.
//
// Ctrl selects deployment targets. Krane publishes one shared headless Service
// per deployment port, named by [ServiceName], and fills it with filtered
// source EndpointSlices that Cilium shares across clusters. Each caller
// deployment gets one ExternalName Service per alias, named by
// [ConnectionServiceName], whose ExternalName is the [ServiceHost] of the
// shared Service it currently resolves to. [TargetDeploymentAnnotation]
// records that deployment. Stock CoreDNS answers from these Services.
//
// Each source endpoint's hostname is the region of the cluster that publishes
// it. For every topology region, a caller also gets one ExternalName Service
// named by [RegionalConnectionServiceName] that resolves only the endpoints
// with that hostname.
//
// DNS visibility is not authorization. Cilium policies grant traffic.
//
// [AppendReadyAddresses] defines which endpoints count as ready before Krane
// moves an alias to a new deployment. This package does not perform I/O.
package privatenetwork
