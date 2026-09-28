// Package undns is the regional DNS resolver for directed app bindings. It
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
// Directed bindings grant one caller deployment access to one alias. The
// resolver doesn't infer reverse access and doesn't fall back to bindings for
// another deployment, app, project, or workspace.
//
// Ctrl also publishes unkey-replicas.unkey.internal for every active deployment,
// without a stored app binding. It resolves that deployment's ready replicas
// across regions, excluding other versions of the app. When private DNS is
// configured, Krane injects this hostname as UNKEY_REPLICA_HOST and grants
// unicast TCP and UDP connectivity between those replicas.
//
// # Answers
//
// A queries return unique, ready private IPv4 endpoints for the bound target
// deployment in random order, so clients that take the first address spread
// across replicas. AAAA queries return NODATA, and unsupported types return
// NOTIMP. Unknown aliases and multi-label names return NXDOMAIN. A known but
// unresolved or unusable binding returns SERVFAIL. Private queries also return
// SERVFAIL until every discovery watch is healthy and activation has run.
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
// Krane publishes each binding as a ConfigMap labeled managed-by=krane,
// component=private-dns, workspace ID, project ID, target app ID, caller
// deployment ID, and binding ID. Data contains appSlug (the binding alias),
// deploymentId, serviceName, and a positive monotonic revision. deploymentId
// and serviceName are both empty while a known binding has no target. The
// resolver keys bindings by workspace, project, caller deployment, and alias.
// Duplicate bindings for a key return SERVFAIL.
//
// A target discovery Service is shared by bindings. It must be headless, use
// publishNotReadyAddresses=false, and have managed-by, component, workspace,
// project, target app, and target deployment labels. Caller and binding labels
// aren't accepted as Service identity. The Service and ConfigMap must share a
// namespace. An EndpointSlice contributes addresses only when its controller
// reference matches the Service name and UID.
//
// # Activation
//
// A higher binding revision switches only after its target has a ready
// endpoint. Until then, a running resolver keeps the prior target for the same
// binding identity. A higher revision can explicitly roll back to an older
// deployment. Once a target activates, losing its endpoints returns SERVFAIL
// instead of reviving an earlier target.
//
// Changing the binding ID, caller deployment, alias, target app, object name,
// object UID, or namespace revokes prior activation. Deleting a binding makes
// the alias unknown. Publishing an unresolved binding revokes prior activation
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
// unkey_dns_discovery_ready reports private discovery health separately.
// Restrict resolver access to cluster workloads; unknown callers can forward
// public queries and are not authenticated by DNS.
//
// Local watches renew every watch_timeout. A reported watch error closes
// private discovery immediately; a silent stall does so after twice watch_timeout.
// Endpoint availability reflects Kubernetes and Cilium EndpointSlice state,
// not a direct health check of a remote Pod. DNS doesn't provide transport
// isolation, encryption, authentication, IPv6 answers, broadcast, or multicast.
package undns
