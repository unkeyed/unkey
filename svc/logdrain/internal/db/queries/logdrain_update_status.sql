-- UpdateLogdrainStatus sets only the status. Unlike the dashboard, it keeps
-- the lease and failure state so tests can isolate the status guards. Only
-- tests call it.
-- name: UpdateLogdrainStatus :exec
UPDATE logdrains
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(id);
