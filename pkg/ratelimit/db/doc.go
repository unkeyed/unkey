// Package db contains the queries of the ratelimit_state table, which holds the
// shared state of the global rate limits.
//
// Each row is one session of one pod on one limit. The state of a limit is
//
//	reset_at = MAX over rows r of
//	           (r.anchor_at_ns + SUM(consumed_ns of rows anchored at or after r))
//
// No row depends on another row. consumed_ns only grows, so
// [Querier.UpsertSession] keeps the greater value and a retried or reordered
// write cannot undo a newer one. anchor_at_ns and compact_after_ms are set
// when the row is created and never change.
//
// Every query is bounded. Callers pass the bound as an argument and must check
// whether a result reached it.
//
// Callers own the connections. Writes and cleanup go to the primary, and
// reads for decisions go to a replica:
//
//	sessions, err := db.Query.ListSessions(ctx, replica, db.ListSessionsParams{
//		Fingerprint: fingerprint,
//		Limit:       sessionsMax,
//	})
package db
