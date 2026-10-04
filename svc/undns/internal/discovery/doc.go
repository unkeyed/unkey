// Package discovery keeps the resolver's read-only view of Krane's published
// private DNS objects and answers caller identity and endpoint lookups from
// local informer caches. Lookups never call the Kubernetes API, Ctrl, or a
// database.
//
// # Watches
//
// [Catalog] watches Krane deployment Pods and objects labeled
// component=private-dns in every namespace, plus the topology ConfigMap in
// kube-system. Cilium copies those labels onto the EndpointSlices it imports
// from other clusters. Every watch renews after watch_timeout. A reported
// watch error marks it stale immediately, and a silent stall marks it stale
// after twice watch_timeout, because informers never clear HasSynced after a
// lost watch. [Catalog.Ready] covers the Pod, connection, Service, and
// EndpointSlice watches; [Catalog.IdentityReady] covers only Pods. The topology
// watch is not a readiness gate: unknown topology uses known cluster-local
// endpoints for local-first, or all ready endpoints if there are none.
//
// # Discovery objects
//
// Krane publishes each connection as a ConfigMap labeled managed-by=krane,
// component=private-dns, workspace ID, project ID, target app ID, caller
// deployment ID, and connection ID. Its data is written by
// privatenetwork.Encode: appSlug (the connection alias), deploymentId,
// serviceName, a positive monotonic revision, and schemaVersion. deploymentId
// and serviceName are both empty while a known connection has no target.
// Connections are keyed by workspace, project, caller deployment, and alias.
//
// A target discovery Service is shared by connections. It must be headless,
// use publishNotReadyAddresses=false, and have managed-by, component,
// workspace, project, target app, and target deployment labels. Caller and
// connection labels aren't accepted as Service identity. The Service and
// ConfigMap must share a namespace, which must also be the caller's namespace.
// Krane publishes a source EndpointSlice containing only ready, non-terminating
// addresses; its controller reference must match the Service name and UID.
// Imported slices are attributed to regions with Cilium's source-cluster label
// and the topology ConfigMap, not with Service region labels.
//
// # Failures and connection states
//
// Lookup failures are an [*Error] whose [Reason] is a stable metric label;
// [ReasonOf] extracts it. Once per second, after the caches synchronize,
// [Catalog.Run] recomputes the answer every published connection would get
// when discovery changed or a Service retirement deadline passed, publishes
// the counts as unkey_dns_connections, and logs each state change once with
// the connection and caller IDs that metrics omit. The same loop publishes
// unkey_dns_discovery_ready and unkey_dns_discovery_watch_healthy, so those
// gauges have one second resolution.
package discovery
