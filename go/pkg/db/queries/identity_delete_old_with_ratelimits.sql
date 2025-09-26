-- name: DeleteOldIdentityWithRatelimits :exec
DELETE i, rl
FROM identities i
LEFT JOIN ratelimits rl ON rl.identity_id = i.id
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND (i.id = sqlc.arg(identity) OR i.external_id = sqlc.arg(identity))
  AND i.deleted = true;
