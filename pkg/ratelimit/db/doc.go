// Package db contains the queries of the ratelimit_state table, which holds the
// shared state of the global rate limits.
//
// Each row is the consumed time of one pod in one time bucket of one limit,
// and the GCRA reset_at of those requests alone. Bucket edges are shared by
// all pods. The state of a limit is
//
//	bucket_reset_at = MAX(bucket_start_ns + SUM(consumed_ns), MAX(reset_at_ns))
//	                  over the rows of one bucket
//	reset_at        = MAX over buckets b of
//	                  (bucket_reset_at of b + SUM(consumed_ns of newer buckets))
//
// Both values only grow, so [Querier.UpsertRow] keeps the greater value and a
// retried or reordered write cannot undo a newer one. MySQL refuses writes
// to a row once its compact_after_ns has passed by the MySQL clock. Cleanup
// then merges closed rows into the base row, which has bucket_start_ns 0 and
// an empty pod_id.
//
// Every query is bounded. Callers pass the bound as an argument and must check
// whether a result reached it.
//
// Callers own the connections. Writes and cleanup go to the primary, and
// reads for decisions go to a replica:
//
//	buckets, err := db.Query.ListBuckets(ctx, replica, db.ListBucketsParams{
//		PodID:       podID,
//		Fingerprint: fingerprint,
//		Limit:       bucketsMax,
//	})
package db
