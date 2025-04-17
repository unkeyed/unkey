-- name: FindRatelimitOverrideMatches :many
SELECT * FROM ratelimit_overrides
WHERE
    workspace_id = sqlc.arg(workspace_id)
    AND namespace_id = sqlc.arg(namespace_id)
    AND sqlc.arg(identifier) LIKE
          REPLACE(
            REPLACE(identifier, '*', '%'), -- Replace * with % wildcard
            '_', '\\_'                              -- Escape underscore literals
          );
