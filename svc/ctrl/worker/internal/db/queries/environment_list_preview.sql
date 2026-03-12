-- name: ListPreviewEnvironments :many
SELECT *
FROM environments
WHERE slug = 'preview'
AND pk > sqlc.arg(pagination_cursor)
ORDER BY pk ASC
LIMIT ?;
