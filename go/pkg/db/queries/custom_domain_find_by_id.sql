-- name: FindCustomDomainById :one
SELECT *
FROM custom_domains
WHERE id = sqlc.arg(id);
