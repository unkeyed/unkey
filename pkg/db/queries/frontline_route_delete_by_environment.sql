-- name: DeleteFrontlineRoutesByEnvironmentId :execresult
-- Paginated delete: caller loops until RowsAffected < batch_limit.
-- A single unbounded DELETE could exceed transaction/replication size
-- limits for environments with many routes; paginating with the same
-- WHERE clause means each tick deletes a bounded number of rows and
-- the loop naturally terminates when no rows remain.
DELETE FROM frontline_routes
WHERE environment_id = sqlc.arg(environment_id)
LIMIT ?;
