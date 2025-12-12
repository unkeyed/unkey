-- name: ListDesiredSentinels :many
SELECT
    sqlc.embed(sentinels),
    sqlc.embed(workspaces)
FROM `sentinels`
INNER JOIN `workspaces` ON sentinels.workspace_id = workspaces.id
WHERE (sqlc.arg(region) = '' OR region = sqlc.arg(region))
    AND desired_state = sqlc.arg(desired_state)
    AND sentinels.id > sqlc.arg(pagination_cursor)
ORDER BY sentinels.id ASC
LIMIT ?;
