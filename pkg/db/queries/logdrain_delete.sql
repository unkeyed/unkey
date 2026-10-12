-- name: DeleteLogdrain :exec
-- Caller holds the drain lock and inserts its audit event in this transaction.
-- Worker state updates cannot recreate a deleted drain.
DELETE FROM logdrains WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);
