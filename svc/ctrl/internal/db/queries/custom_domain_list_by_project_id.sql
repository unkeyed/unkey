-- name: ListCustomDomainsByProjectID :many
SELECT *
FROM custom_domains
WHERE project_id = sqlc.arg(project_id)
ORDER BY created_at DESC;
