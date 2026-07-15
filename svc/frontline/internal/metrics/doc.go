// Package metrics holds the cross-cutting per-request metrics frontline
// emits from its observability middleware: a unified outcome counter
// labelled by URN, and a saturation gauge.
//
// Subpackage-local metrics (routing decisions, upstream timing, hops, TLS
// handshakes, etc.) live in the package that emits them, not here.
//
// Region and environment are external labels on the per-region Prometheus,
// so queries can `sum by (region)` without paying the cardinality cost on
// every emitted series.
package metrics
