// Package db contains the queries of the ratelimit_state table, which holds the
// shared state of the global rate limits.
//
// Each row is the scaled sum of one pod in one period of one limit:
//
//	scaled_sum = SUM(cost × e^((t − period_start_ms) / duration))
//
// over the requests at time t that the pod allowed in the period. Period
// edges are shared by all pods. The level of a limit at now is
//
//	level = SUM(scaled_sum × e^((period_start_ms − now) / duration))
//
// The sum only grows, so [Querier.UpsertRow] keeps the greater value and a
// retried or reordered write cannot undo a newer one. Cleanup deletes rows
// whose delete_after_ms has passed by the MySQL clock.
//
// Every query is bounded. Callers pass the bound as an argument.
//
// Callers own the connections. Writes and cleanup go to the primary, and
// reads for decisions go to a replica:
//
//	periods, err := db.Query.ListPeriods(ctx, replica, db.ListPeriodsParams{
//		PodID:       podID,
//		Fingerprint: fingerprint,
//		Limit:       periodsMax,
//	})
package db
