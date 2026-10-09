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
// created with private networking, without a stored app connection. Every new
// non-skipped deployment saves private_networking; legacy deployments that saved
// false remain disabled. A deployment with zero directed connections can still
// use its self DNS name. The name resolves that deployment's ready replicas
// across regions, excluding other versions of the app. Krane injects it as
// UNKEY_PRIVATE_DOMAIN, points the deployment's Pods at undns, and grants
// unicast TCP and UDP connectivity between those replicas. Other deployments
// keep cluster DNS. Connection names can't reuse the caller app's slug or start
// with unkey.
//
// # Answers
//
// A queries return unique, ready private IPv4 endpoints for the bound target
// deployment in random order, so clients that take the first address spread
// across replicas. AAAA queries return NODATA, and unsupported types return
// NOTIMP. Unknown aliases and invalid selectors return NXDOMAIN. A known but
// unresolved or unusable connection, or duplicate connections for one alias,
// return SERVFAIL. Private queries also return SERVFAIL until every discovery
// watch is synchronized and healthy.
//
// <region>.<alias>.unkey.internal selects only ready endpoints in that region;
// an empty result returns SERVFAIL. local-first.<alias>.unkey.internal selects
// the caller's region, falling back to all ready endpoints only if none match.
// Both filter after authorization. Ordinary names, including same-deployment
// peer discovery, still return all ready endpoints.
//
// Public responses use one shared upstream client and a TTL cache holding at
// most forward_cache_entries responses, which must be at least 10. An upstream
// timeout, transport failure, or SERVFAIL can use an expired positive answer
// for at most 30 seconds past expiry with TTL 0. Negative and private answers
// are never served stale. Forward quotas still apply to cache hits. Upstream
// TTLs with the most significant bit set are treated as zero (RFC 2181).
//
// # Published authority
//
// The currently observed connection ConfigMap is the sole authority for a
// connection. Switching it changes the target immediately, including an
// explicit rollback; revision remains validated observation data and does not
// gate answers. If the published target is unresolved or has no ready
// endpoints, queries return SERVFAIL rather than retaining an older target.
// Deleting the ConfigMap makes the alias unknown. A restarted and a
// long-running resolver therefore answer from the same published state.
//
// Services with an expired privatenetwork retire-after annotation are rejected.
// Clients can cache answers for their TTL or longer, and existing connections
// aren't moved.
//
// # Load shedding
//
// Local answers and upstream forwards use separate limits. queries_in_flight
// bounds local work, and forwards_in_flight bounds upstream forwards, of which
// one caller workspace can hold at most forwards_per_workspace. A full limit
// answers SERVFAIL immediately and increments unkey_dns_shed_total.
//
// Callers without a fresh Pod identity share one forwards_unidentified quota
// within the global forward limit, regardless of source IP. Its default is 32.
// During a Pod watch outage, all public callers share this quota; per-workspace
// fairness resumes when identity discovery recovers. These are concurrency
// limits, not packet-rate limits.
//
// # Operation
//
// Run undns with a regional CoreDNS upstream and expose its listen address on
// UDP and TCP while preserving Pod source IPs. A shared caching proxy must not
// sit in front of undns because answers depend on caller deployment identity.
// Health endpoints are /health/live, /health/ready, and /health/startup on the
// health address. Readiness probes the UDP and TCP listeners, not discovery or
// the upstream resolver, and fails while the process shuts down. A failed TCP
// listener stops the process. On shutdown, accepted TCP requests drain until
// the shutdown timeout.
//
// Prometheus metrics are available at /metrics on the health address.
// unkey_dns_discovery_ready and unkey_dns_discovery_watch_healthy report private
// discovery health separately, refreshed once per second.
// unkey_dns_queries_total labels every response with its path and a bounded
// reason, such as no_ready_endpoints or upstream_timeout, and
// unkey_dns_connections reports the answer each published connection would
// get. Logs carry the connection and caller IDs that metrics omit: connection
// state changes are logged once per change, and query failures are sampled
// once per reason per minute. Restrict resolver access to cluster workloads;
// unknown callers can forward public queries and are not authenticated by DNS.
//
// A reported watch error closes private discovery immediately; a silent stall
// does so after twice watch_timeout. Endpoint availability reflects Kubernetes
// and Cilium EndpointSlice state, not a direct health check of a remote Pod.
// DNS doesn't provide transport isolation, encryption, authentication, IPv6
// answers, broadcast, or multicast.
package undns
