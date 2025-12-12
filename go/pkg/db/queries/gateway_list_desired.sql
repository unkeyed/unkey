-- name: ListDesiredGateways :many
SELECT
    sqlc.embed(gateways),
    sqlc.embed(workspaces)
FROM `gateways`
INNER JOIN `workspaces` ON gateways.workspace_id = workspaces.id
WHERE (sqlc.arg(region) = '' OR region = sqlc.arg(region))
    AND desired_state = sqlc.arg(desired_state)
    AND gateways.id > sqlc.arg(pagination_cursor)
ORDER BY gateways.id ASC
LIMIT ?;
