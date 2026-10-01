// Package undns is the regional DNS resolver for directed app connections. It
// answers <alias>.unkey.internal from local Kubernetes objects and forwards
// every other name to regional CoreDNS. Queries never call Ctrl or a database.
//
// # Caller identity
//
// A private query source IP must identify exactly one Running, non-terminating,
// non-host-networked Krane deployment Pod. The Pod must have workspace,
// project, app, environment, environment kind, and deployment identity labels.
// Failed and Succeeded Pods don't retain an identity after their IP is reused.
// Invalid callers receive REFUSED for private names. An unhealthy Pod watch
// makes private queries fail with SERVFAIL. Public forwarding does not require
// synchronized discovery or a known caller.
//
// Directed connections grant one caller deployment access to one alias. The
// resolver doesn't infer reverse access and doesn't fall back to connections for
// another deployment, app, project, or workspace.
//
// Ctrl also publishes <app-slug>.unkey.internal for every active deployment
// created with private networking, without a stored app connection. Ctrl decides
// that once when it creates the deployment: the workspace needs an app connection
// to another app and the private-networking feature flag. The name resolves
// that deployment's ready replicas across regions, excluding other versions of
// the app. Krane injects it as UNKEY_PRIVATE_DOMAIN, points the deployment's
// Pods at undns, and grants unicast TCP and UDP connectivity between those
// replicas. Other deployments keep cluster DNS. Connection names can't reuse the
// caller app's slug or start with unkey.
//
// # Answers
//
// A queries return unique, ready private IPv4 endpoints for the bound target
// deployment in random order, so clients that take the first address spread
// across replicas. AAAA queries return NODATA, and unsupported types return
// NOTIMP. Unknown aliases and invalid selectors return NXDOMAIN. A known but
// unresolved or unusable connection returns SERVFAIL. Private queries also return
// SERVFAIL until every discovery watch is healthy and activation has run.
//
// <region>.<alias>.unkey.internal selects only ready endpoints in that region;
// an empty result returns SERVFAIL. local-first.<alias>.unkey.internal selects
// the caller's region, falling back to all ready endpoints only if none match.
// Both filter after revision activation and authorization. Ordinary names,
// including same-deployment peer discovery, still return all ready endpoints.
// The separate topology watch is not a readiness gate: unknown topology uses
// known cluster-local endpoints for local-first, or all ready endpoints if empty.
// Imported slices use Cilium's source-cluster label, not Service region labels.
//
// # Load shedding
//
// Local answers and upstream forwards use separate limits. queries_in_flight
// bounds local work. A forward releases that slot before contacting the
// upstream and takes one of forwards_in_flight instead, and one caller
// workspace can hold at most forwards_per_workspace of those. A full limit
// answers SERVFAIL immediately and increments unkey_dns_shed_total.
//
// Callers without a fresh Pod identity share one forwards_unidentified quota
// within the global forward limit, regardless of source IP. Its default is 32.
// During a Pod watch outage, all public callers share this quota; per-workspace
// fairness resumes when identity discovery recovers. These are concurrency
// limits, not packet-rate limits.
//
// # Discovery objects
//
// Krane publishes each connection as a ConfigMap labeled managed-by=krane,
// component=private-dns, workspace ID, project ID, target app ID, caller
// deployment ID, and connection ID. Data contains appSlug (the connection alias),
// deploymentId, serviceName, and a positive monotonic revision. deploymentId
// and serviceName are both empty while a known connection has no target. The
// resolver keys connections by workspace, project, caller deployment, and alias.
// Duplicate connections for a key return SERVFAIL.
//
// A target discovery Service is shared by connections. It must be headless, use
// publishNotReadyAddresses=false, and have managed-by, component, workspace,
// project, target app, and target deployment labels. Caller and connection labels
// aren't accepted as Service identity. The Service and ConfigMap must share a
// namespace. An EndpointSlice contributes addresses only when its controller
// reference matches the Service name and UID.
//
// # Activation
//
// A higher connection revision switches only after its target has a ready
// endpoint. Until then, a running resolver keeps the prior target for the same
// connection identity. A higher revision can explicitly roll back to an older
// deployment. Once a target activates, losing its endpoints returns SERVFAIL
// instead of reviving an earlier target.
//
// Changing the connection ID, caller deployment, alias, target app, object name,
// object UID, or namespace revokes prior activation. Deleting a connection makes
// the alias unknown. Publishing an unresolved connection revokes prior activation
// and returns SERVFAIL, so a removed target can't be retained or resurrected.
// A restart can activate the durable target in the ConfigMap independently of
// discovery objects staged for a future revision.
//
// Services with an expired privatenetwork retire-after annotation are rejected.
// Clients can cache answers for their TTL or longer, and existing connections
// aren't moved.
//
// # Operation
//
// Run undns with a regional CoreDNS upstream and expose its listen address on
// UDP and TCP while preserving Pod source IPs. A shared caching proxy must not
// sit in front of undns because answers depend on caller deployment identity.
// Health endpoints are /health/live, /health/ready, and /health/startup.
// Readiness checks the UDP and TCP listeners, not private discovery or the
// upstream resolver. Prometheus metrics are available at /metrics;
// unkey_dns_discovery_ready and unkey_dns_discovery_watch_healthy report
// private discovery health separately. unkey_dns_queries_total labels every
// response with its path and a bounded reason, such as no_ready_endpoints or
// upstream_timeout, and unkey_dns_connections reports the answer each published
// connection would get. Logs carry the connection and caller IDs that metrics omit:
// connection state changes are logged once per change, and query failures are
// sampled once per reason per minute.
// Restrict resolver access to cluster workloads; unknown callers can forward
// public queries and are not authenticated by DNS.
//
// Watches select only Krane deployment Pods and objects labeled
// component=private-dns; Cilium copies those labels onto the EndpointSlices it
// imports from other clusters. Local watches renew every watch_timeout. A reported watch error closes
// private discovery immediately; a silent stall does so after twice watch_timeout.
// Endpoint availability reflects Kubernetes and Cilium EndpointSlice state,
// not a direct health check of a remote Pod. DNS doesn't provide transport
// isolation, encryption, authentication, IPv6 answers, broadcast, or multicast.
package undns
