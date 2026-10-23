-- name: LockCustomDomain :one
SELECT id FROM custom_domains WHERE id = sqlc.arg(id)
LOCK IN SHARE MODE;
