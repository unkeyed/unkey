-- name: ListPreviewEnvironments :many
SELECT *
FROM environments
WHERE kind = 'preview'
AND pk > sqlc.arg(pagination_cursor)
ORDER BY pk ASC
LIMIT ?;
