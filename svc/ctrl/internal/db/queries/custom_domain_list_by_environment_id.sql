-- name: ListCustomDomainsByEnvironmentID :many
SELECT *
FROM custom_domains
WHERE environment_id = sqlc.arg(environment_id);
