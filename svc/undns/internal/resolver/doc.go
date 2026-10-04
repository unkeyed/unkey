// Package resolver answers DNS queries for undns. [Resolver] answers names
// below unkey.internal from a [Catalog] scoped to the querying Pod and forwards
// every other name to the regional upstream. It doesn't own sockets; the
// server package frames and delivers its messages.
//
// # Private answers
//
// The source IP must identify a caller while the Pod watch is fresh, otherwise
// private names get REFUSED, or SERVFAIL during a Pod watch outage. Private
// names also get SERVFAIL until discovery is ready. A queries return the ready
// endpoints in random order with ttl_seconds; AAAA queries return NODATA and
// other types NOTIMP. Unknown names return NXDOMAIN with the zone SOA, and a
// known name that can't be answered returns SERVFAIL. Private answers are
// never cached.
//
// # Forwarding
//
// Public queries are rebuilt from the question alone, so client EDNS options
// such as client subnet never reach the upstream, and a truncated UDP answer is
// retried over TCP. Responses are cached on top of pkg/cache, keyed by
// lower-cased name, type, class, and the RD bit. A cached response lives for
// its minimum record TTL, or for min(SOA TTL, SOA MINIMUM) when negative, and
// its TTLs decay as it ages. Truncated responses aren't cached, and a zero-TTL
// success or NXDOMAIN removes the previous entry. An upstream timeout,
// transport error, or SERVFAIL can use an expired positive answer for at most
// 30 seconds past expiry with TTL 0. forward_cache_entries, at least 10,
// bounds the number of cached responses.
//
// # Load shedding
//
// queries_in_flight bounds local work. A forward releases that slot before
// contacting the upstream and takes one of forwards_in_flight instead, plus a
// slot of its workspace's forwards_per_workspace quota, or of the shared
// forwards_unidentified quota when the caller has no fresh identity. Quotas
// also apply to cache hits. A full limit answers SERVFAIL immediately and
// increments unkey_dns_shed_total.
package resolver
