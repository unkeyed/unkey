-- name: CountAppsById :one
-- CountAppsById counts apps with an id. Only tests use this query.
SELECT COUNT(*)
FROM `apps`
WHERE id = sqlc.arg(id);
